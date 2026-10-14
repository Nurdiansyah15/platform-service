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

	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	"platform-service/internal/app/usecase/solution"
	"platform-service/internal/app/usecase/system"
	"platform-service/internal/config"
	i18n "platform-service/internal/infrastructure/i18n"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/router"

	"github.com/gin-gonic/gin"
)

type Options struct {
	Config *config.Config
	DB     *sql.DB
	Logger *slog.Logger

	// Authenticator — nil berarti semua token ditolak (sampai verifier JWT
	// tersedia di Fase 2–4). Test boleh meng-override dengan fake.
	Authenticator port.TokenAuthenticator
}

// NewEngine membangun engine HTTP siap pakai.
func NewEngine(opts Options) (*gin.Engine, error) {
	if opts.Config == nil || opts.DB == nil || opts.Logger == nil {
		return nil, errors.New("bootstrap: Config, DB, dan Logger wajib diisi")
	}

	bundle, err := i18n.Load(opts.Config.App.LocalesDir)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: load locales: %w", err)
	}
	httperror.InitBundle(bundle)
	httperror.RegisterValidatorTagNames()

	authn := opts.Authenticator
	if authn == nil {
		authn = rejectAllAuthenticator{}
	}

	// ── Persistence ─────────────────────────────────────────────────────────
	healthProbe := persistence.NewPostgresHealthProbe(opts.DB)
	solutionRepo := persistence.NewPostgresSolutionRepository(opts.DB)
	solutionQuery := persistence.NewPostgresSolutionQuery(opts.DB)

	// ── Usecases ────────────────────────────────────────────────────────────
	systemUC := system.NewSystemUseCase(system.SystemDeps{
		DB:             healthProbe,
		DeploymentMode: string(opts.Config.App.DeploymentMode),
		Logger:         opts.Logger,
	})
	solutionUC := solution.NewSolutionUseCase(solution.SolutionDeps{Repo: solutionRepo, Query: solutionQuery})

	// ── Handlers & router ───────────────────────────────────────────────────
	return router.Setup(router.Deps{
		Config:        opts.Config,
		Logger:        opts.Logger,
		Authenticator: authn,
		Handlers: router.Handlers{
			System:   api.NewSystemHandler(systemUC),
			Solution: api.NewSolutionHandler(solutionUC),
		},
	}), nil
}

var errAuthenticatorUnavailable = errors.New("token authenticator belum dikonfigurasi")

// rejectAllAuthenticator dipakai sampai verifier JWT tersedia (Fase 2–4).
type rejectAllAuthenticator struct{}

func (rejectAllAuthenticator) Authenticate(context.Context, string) (*principal.Principal, error) {
	return nil, errAuthenticatorUnavailable
}
