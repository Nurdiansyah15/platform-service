// Package bootstrap merangkai seluruh dependency aplikasi
// (repository → usecase → handler → router). Dipakai bersama oleh cmd/app dan
// internal/testhelper agar route & wiring yang diuji sama persis dengan produksi.
package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	"platform-service/internal/app/usecase/auth"
	"platform-service/internal/app/usecase/signingkey"
	"platform-service/internal/app/usecase/solution"
	"platform-service/internal/app/usecase/system"
	"platform-service/internal/app/usecase/user"
	"platform-service/internal/config"
	"platform-service/internal/infrastructure/crypto"
	i18n "platform-service/internal/infrastructure/i18n"
	"platform-service/internal/infrastructure/jwttoken"
	"platform-service/internal/infrastructure/keystore"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/router"
)

const startupTimeout = 30 * time.Second

type Options struct {
	Config *config.Config
	DB     *sql.DB
	Logger *slog.Logger

	// KeyStore — nil berarti FileKeyStore di Config.Auth.KeyStoreDir.
	KeyStore port.KeyStore

	// PasswordHasher — nil berarti bcrypt dengan cost default. Test memakai cost minimum.
	PasswordHasher port.PasswordHasher

	// ExtraAuthenticator dicoba sebelum verifier JWT (mis. FakeAuthenticator di test).
	// Produksi membiarkannya nil.
	ExtraAuthenticator port.TokenAuthenticator
}

// App adalah hasil wiring yang siap dijalankan.
type App struct {
	Engine      *gin.Engine
	TokenIssuer port.TokenIssuer
	// Users dipakai seeder/test untuk menyiapkan platform admin.
	Users *user.UserUseCase
}

// New membangun aplikasi lengkap dan menjalankan tugas startup
// (memastikan signing key active tersedia).
func New(ctx context.Context, opts Options) (*App, error) {
	if opts.Config == nil || opts.DB == nil || opts.Logger == nil {
		return nil, errors.New("bootstrap: Config, DB, dan Logger wajib diisi")
	}
	cfg := opts.Config

	bundle, err := i18n.Load(cfg.App.LocalesDir)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: load locales: %w", err)
	}
	httperror.InitBundle(bundle)
	httperror.RegisterValidatorTagNames()

	hasher := opts.PasswordHasher
	if hasher == nil {
		hasher = crypto.NewBcryptHasher(0)
	}

	keyStore := opts.KeyStore
	if keyStore == nil {
		if keyStore, err = keystore.NewFileKeyStore(cfg.Auth.KeyStoreDir); err != nil {
			return nil, fmt.Errorf("bootstrap: %w", err)
		}
	}

	// ── Persistence ─────────────────────────────────────────────────────────
	healthProbe := persistence.NewPostgresHealthProbe(opts.DB)
	solutionRepo := persistence.NewPostgresSolutionRepository(opts.DB)
	solutionQuery := persistence.NewPostgresSolutionQuery(opts.DB)
	signingKeyRepo := persistence.NewPostgresSigningKeyRepository(opts.DB)
	userRepo := persistence.NewPostgresUserRepository(opts.DB)
	sessionRepo := persistence.NewPostgresAuthSessionRepository(opts.DB)

	// ── Token ───────────────────────────────────────────────────────────────
	keyRing := jwttoken.NewKeyRing(signingKeyRepo, keyStore)
	tokenIssuer := jwttoken.NewIssuer(keyRing, cfg.App.IssuerURL, cfg.Auth.AccessTokenTTL)

	// ── Usecases ────────────────────────────────────────────────────────────
	systemUC := system.NewSystemUseCase(system.SystemDeps{
		DB:             healthProbe,
		DeploymentMode: string(cfg.App.DeploymentMode),
		Logger:         opts.Logger,
	})
	solutionUC := solution.NewSolutionUseCase(solution.SolutionDeps{Repo: solutionRepo, Query: solutionQuery})
	signingKeyUC := signingkey.NewSigningKeyUseCase(signingkey.SigningKeyDeps{
		Repo:     signingKeyRepo,
		KeyStore: keyStore,
		Logger:   opts.Logger,
	})
	userUC := user.NewUserUseCase(user.UserDeps{Repo: userRepo, Hasher: hasher})
	authUC, err := auth.NewAuthUseCase(auth.AuthDeps{
		Users:           userRepo,
		Solutions:       solutionRepo,
		Sessions:        sessionRepo,
		Hasher:          hasher,
		Tokens:          tokenIssuer,
		RefreshTokenTTL: cfg.Auth.RefreshTokenTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("bootstrap: auth usecase: %w", err)
	}

	// ── Authenticator ───────────────────────────────────────────────────────
	// JWT diverifikasi dulu, lalu token user dicek ke sesi & user di DB.
	var authn port.TokenAuthenticator = sessionAwareAuthenticator{
		token:       jwttoken.NewAuthenticator(keyRing, cfg.App.IssuerURL),
		resolveUser: authUC.ResolveUserPrincipal,
	}
	if opts.ExtraAuthenticator != nil {
		authn = chainAuthenticator{opts.ExtraAuthenticator, authn}
	}

	// ── Startup ─────────────────────────────────────────────────────────────
	startupCtx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()
	if err := signingKeyUC.EnsureActiveKey(startupCtx); err != nil {
		return nil, fmt.Errorf("bootstrap: ensure signing key: %w", err)
	}

	// ── Handlers & router ───────────────────────────────────────────────────
	engine := router.Setup(router.Deps{
		Config:        cfg,
		Logger:        opts.Logger,
		Authenticator: authn,
		Handlers: router.Handlers{
			System:   api.NewSystemHandler(systemUC),
			Solution: api.NewSolutionHandler(solutionUC),
			JWKS:     api.NewJWKSHandler(signingKeyUC),
			Auth:     api.NewAuthHandler(authUC),
			User:     api.NewUserHandler(userUC),
		},
	})

	return &App{Engine: engine, TokenIssuer: tokenIssuer, Users: userUC}, nil
}

// sessionAwareAuthenticator memverifikasi JWT lalu memvalidasi token user ke DB
// (sesi aktif, user active, IsPlatformAdmin). Token service diteruskan apa adanya
// sampai validasi solution client tersedia (Fase 4).
type sessionAwareAuthenticator struct {
	token       port.TokenAuthenticator
	resolveUser func(ctx context.Context, p *principal.Principal) (*principal.Principal, error)
}

func (a sessionAwareAuthenticator) Authenticate(ctx context.Context, token string) (*principal.Principal, error) {
	p, err := a.token.Authenticate(ctx, token)
	if err != nil {
		return nil, err
	}
	if p.IsUser() {
		return a.resolveUser(ctx, p)
	}
	return p, nil
}

// chainAuthenticator mencoba authenticator berurutan; yang pertama berhasil dipakai.
type chainAuthenticator []port.TokenAuthenticator

func (c chainAuthenticator) Authenticate(ctx context.Context, token string) (*principal.Principal, error) {
	var errs []error
	for _, a := range c {
		p, err := a.Authenticate(ctx, token)
		if err == nil {
			return p, nil
		}
		errs = append(errs, err)
	}
	return nil, errors.Join(errs...)
}
