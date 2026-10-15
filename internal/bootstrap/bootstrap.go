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
	"platform-service/internal/app/usecase/addon"
	"platform-service/internal/app/usecase/auth"
	"platform-service/internal/app/usecase/authz"
	"platform-service/internal/app/usecase/benefit"
	"platform-service/internal/app/usecase/entitlement"
	"platform-service/internal/app/usecase/organization"
	"platform-service/internal/app/usecase/permission"
	"platform-service/internal/app/usecase/plan"
	"platform-service/internal/app/usecase/role"
	"platform-service/internal/app/usecase/signingkey"
	"platform-service/internal/app/usecase/solution"
	"platform-service/internal/app/usecase/solutionclient"
	"platform-service/internal/app/usecase/subscription"
	"platform-service/internal/app/usecase/system"
	"platform-service/internal/app/usecase/tenant"
	"platform-service/internal/app/usecase/usage"
	"platform-service/internal/app/usecase/user"
	"platform-service/internal/config"
	authzservice "platform-service/internal/domain/authz/service"
	entservice "platform-service/internal/domain/entitlement/service"
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
	clientRepo := persistence.NewPostgresSolutionClientRepository(opts.DB)
	tenantRepo := persistence.NewPostgresTenantRepository(opts.DB)
	memberRepo := persistence.NewPostgresTenantMemberRepository(opts.DB)
	tenantQuery := persistence.NewPostgresTenantQuery(opts.DB)
	companyRepo := persistence.NewPostgresCompanyRepository(opts.DB)
	branchRepo := persistence.NewPostgresBranchRepository(opts.DB)
	orgQuery := persistence.NewPostgresOrganizationQuery(opts.DB)
	benefitRepo := persistence.NewPostgresBenefitRepository(opts.DB)
	benefitQuery := persistence.NewPostgresBenefitQuery(opts.DB)
	planRepo := persistence.NewPostgresPlanRepository(opts.DB)
	planBenefitRepo := persistence.NewPostgresPlanBenefitRepository(opts.DB)
	planQuery := persistence.NewPostgresPlanQuery(opts.DB)
	addonRepo := persistence.NewPostgresAddonRepository(opts.DB)
	addonBenefitRepo := persistence.NewPostgresAddonBenefitRepository(opts.DB)
	addonQuery := persistence.NewPostgresAddonQuery(opts.DB)
	subscriptionRepo := persistence.NewPostgresSubscriptionRepository(opts.DB)
	subscriptionAddonRepo := persistence.NewPostgresSubscriptionAddonRepository(opts.DB)
	snapshotRepo := persistence.NewPostgresSnapshotRepository(opts.DB)
	benefitValueSource := persistence.NewPostgresBenefitValueSource(opts.DB)
	subscriptionQuery := persistence.NewPostgresSubscriptionQuery(opts.DB)
	usageRecordRepo := persistence.NewPostgresUsageRecordRepository(opts.DB)
	tenantUsageRepo := persistence.NewPostgresTenantUsageRepository(opts.DB)
	snapshotReader := persistence.NewPostgresSnapshotReader(opts.DB)
	permissionRepo := persistence.NewPostgresPermissionRepository(opts.DB)
	permissionQuery := persistence.NewPostgresPermissionQuery(opts.DB)
	roleRepo := persistence.NewPostgresRoleRepository(opts.DB)
	rolePermissionRepo := persistence.NewPostgresRolePermissionRepository(opts.DB)
	roleQuery := persistence.NewPostgresRoleQuery(opts.DB)
	assignmentRepo := persistence.NewPostgresRoleAssignmentRepository(opts.DB)
	grantReader := persistence.NewPostgresGrantReader(opts.DB)
	transactor := persistence.NewPostgresTransactor(opts.DB)
	outbox := persistence.NewPostgresEventOutbox()

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
		Tenants:         tenantRepo,
		Members:         memberRepo,
		Subscriptions:   subscriptionRepo,
		Selectable:      subscriptionQuery,
		Hasher:          hasher,
		Tokens:          tokenIssuer,
		RefreshTokenTTL: cfg.Auth.RefreshTokenTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("bootstrap: auth usecase: %w", err)
	}

	clientUC := solutionclient.NewSolutionClientUseCase(solutionclient.SolutionClientDeps{
		Repo:      clientRepo,
		Solutions: solutionRepo,
		Tokens:    tokenIssuer,
		Logger:    opts.Logger,
	})

	tenantDeps := tenant.TenantDeps{
		Tx:      transactor,
		Outbox:  outbox,
		Tenants: tenantRepo,
		Members: memberRepo,
		Users:   userRepo,
		Query:   tenantQuery,
		// AccessPolicy: token service hanya untuk tenant yang subscribe solution-nya.
		Subscriptions: subscriptionRepo,
	}
	tenantUC := tenant.NewTenantUseCase(tenantDeps)
	memberUC := tenant.NewMemberUseCase(tenantDeps)
	orgDeps := organization.OrganizationDeps{
		Tx:        transactor,
		Outbox:    outbox,
		Tenants:   tenantRepo,
		Members:   memberRepo,
		Companies: companyRepo,
		Branches:  branchRepo,
		Query:     orgQuery,
		// AccessPolicy: token service hanya untuk tenant yang subscribe solution-nya.
		Subscriptions: subscriptionRepo,
	}
	companyUC := organization.NewCompanyUseCase(orgDeps)
	branchUC := organization.NewBranchUseCase(orgDeps)
	benefitUC := benefit.NewBenefitUseCase(benefit.BenefitDeps{Repo: benefitRepo, Query: benefitQuery, Solutions: solutionRepo})
	planUC := plan.NewPlanUseCase(plan.PlanDeps{
		Tx:           transactor,
		Plans:        planRepo,
		PlanBenefits: planBenefitRepo,
		Benefits:     benefitRepo,
		Solutions:    solutionRepo,
		Query:        planQuery,
	})
	addonUC := addon.NewAddonUseCase(addon.AddonDeps{
		Tx:            transactor,
		Addons:        addonRepo,
		AddonBenefits: addonBenefitRepo,
		Benefits:      benefitRepo,
		Solutions:     solutionRepo,
		Query:         addonQuery,
	})
	subscriptionUC := subscription.NewSubscriptionUseCase(subscription.SubscriptionDeps{
		Tx:                 transactor,
		Outbox:             outbox,
		Subscriptions:      subscriptionRepo,
		SubscriptionAddons: subscriptionAddonRepo,
		Snapshots:          snapshotRepo,
		BenefitValues:      benefitValueSource,
		Tenants:            tenantRepo,
		Members:            memberRepo,
		Plans:              planRepo,
		Addons:             addonRepo,
		Solutions:          solutionRepo,
		Query:              subscriptionQuery,
	})
	permissionUC := permission.NewPermissionUseCase(permission.PermissionDeps{
		Tx: transactor, Permissions: permissionRepo, Solutions: solutionRepo, Query: permissionQuery,
	})
	roleUC := role.NewRoleUseCase(role.RoleDeps{
		Tx:              transactor,
		Outbox:          outbox,
		Roles:           roleRepo,
		RolePermissions: rolePermissionRepo,
		Permissions:     permissionRepo,
		Solutions:       solutionRepo,
		Tenants:         tenantRepo,
		Members:         memberRepo,
		Subscriptions:   subscriptionRepo,
		Query:           roleQuery,
	})
	authzUC := authz.NewAuthzUseCase(authz.AuthzDeps{
		Tx:            transactor,
		Outbox:        outbox,
		Assignments:   assignmentRepo,
		Grants:        grantReader,
		Scopes:        authzservice.NewScopeValidator(companyRepo, branchRepo),
		Roles:         roleRepo,
		Members:       memberRepo,
		Subscriptions: subscriptionRepo,
		Solutions:     solutionRepo,
	})
	entitlementResolver := entservice.NewResolver(subscriptionRepo, snapshotReader)
	entitlementUC := entitlement.NewEntitlementUseCase(entitlement.EntitlementDeps{
		Benefits:      benefitRepo,
		Solutions:     solutionRepo,
		Tenants:       tenantRepo,
		Members:       memberRepo,
		Subscriptions: subscriptionRepo,
		Resolver:      entitlementResolver,
		TenantUsage:   tenantUsageRepo,
	})
	usageUC := usage.NewUsageUseCase(usage.UsageDeps{
		Tx:            transactor,
		Outbox:        outbox,
		Resolver:      entitlementResolver,
		Records:       usageRecordRepo,
		TenantUsage:   tenantUsageRepo,
		Benefits:      benefitRepo,
		Clients:       clientRepo,
		Subscriptions: subscriptionRepo,
		Members:       memberRepo,
	})

	// ── Authenticator ───────────────────────────────────────────────────────
	// JWT diverifikasi dulu, lalu principal divalidasi ulang ke DB
	// (user: sesi & status user; service: status client & scope terkini).
	var authn port.TokenAuthenticator = dbValidatedAuthenticator{
		token:          jwttoken.NewAuthenticator(keyRing, cfg.App.IssuerURL),
		resolveUser:    authUC.ResolveUserPrincipal,
		resolveService: clientUC.ResolveServicePrincipal,
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
			Client:   api.NewSolutionClientHandler(clientUC),
			Tenant:   api.NewTenantHandler(tenantUC, memberUC),
			Org:      api.NewOrganizationHandler(companyUC, branchUC),
			Benefit:  api.NewBenefitHandler(benefitUC),
			Plan:     api.NewPlanHandler(planUC),
			Addon:    api.NewAddonHandler(addonUC),
			Sub:      api.NewSubscriptionHandler(subscriptionUC),
			Usage:    api.NewUsageHandler(usageUC),
			Ent:      api.NewEntitlementHandler(entitlementUC),
			Perm:     api.NewPermissionHandler(permissionUC),
			Role:     api.NewRoleHandler(roleUC),
			Authz:    api.NewAuthzHandler(authzUC),
		},
	})

	return &App{Engine: engine, TokenIssuer: tokenIssuer, Users: userUC}, nil
}

// dbValidatedAuthenticator memverifikasi JWT lalu memvalidasi principal ke DB,
// sehingga logout, suspend user, dan revoke client langsung berlaku di platform.
type dbValidatedAuthenticator struct {
	token          port.TokenAuthenticator
	resolveUser    func(ctx context.Context, p *principal.Principal) (*principal.Principal, error)
	resolveService func(ctx context.Context, p *principal.Principal) (*principal.Principal, error)
}

func (a dbValidatedAuthenticator) Authenticate(ctx context.Context, token string) (*principal.Principal, error) {
	p, err := a.token.Authenticate(ctx, token)
	if err != nil {
		return nil, err
	}
	switch {
	case p.IsUser():
		return a.resolveUser(ctx, p)
	case p.IsService():
		return a.resolveService(ctx, p)
	default:
		return nil, errors.New("authenticator: jenis principal tidak dikenal")
	}
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
