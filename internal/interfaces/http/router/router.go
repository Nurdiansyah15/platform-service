package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"platform-service/internal/app/port"
	"platform-service/internal/config"
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/middleware"
)

// Handlers berisi semua handler HTTP. Tambah field per domain baru.
type Handlers struct {
	System   *api.SystemHandler
	Solution *api.SolutionHandler
	JWKS     *api.JWKSHandler
	Auth     *api.AuthHandler
	User     *api.UserHandler
	Client   *api.SolutionClientHandler
	Tenant   *api.TenantHandler
	Org      *api.OrganizationHandler
	Benefit  *api.BenefitHandler
}

type Deps struct {
	Config        *config.Config
	Logger        *slog.Logger
	Authenticator port.TokenAuthenticator
	Handlers      Handlers
}

// Routes adalah konteks yang diterima setiap RegisterXxxRoutes.
type Routes struct {
	// V1 adalah group /api/v1 tanpa autentikasi.
	V1 *gin.RouterGroup
	// Authenticated adalah group /api/v1 yang mewajibkan bearer token valid.
	Authenticated *gin.RouterGroup
	// Deployment menyediakan guard LicenseManaged / OnPremOnly / CloudOnly.
	Deployment *middleware.DeploymentGuard
}

// Setup membangun engine Gin lengkap. Dipakai cmd/app dan testhelper — jangan
// mendaftarkan route di tempat lain.
func Setup(d Deps) *gin.Engine {
	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.CORS(d.Config.CORS.AllowOrigins),
		middleware.Locale(),
		middleware.RequestLogger(d.Logger),
		httperror.Middleware(d.Logger),
	)
	r.NoRoute(httperror.NoRoute)

	RegisterSystemRoutes(r, d.Handlers.System)

	v1 := r.Group("/api/v1")
	routes := Routes{
		V1:            v1,
		Authenticated: v1.Group("", middleware.Authenticate(d.Authenticator)),
		Deployment:    middleware.NewDeploymentGuard(d.Config.App.DeploymentMode),
	}
	registerDomainRoutes(routes, d.Handlers)

	return r
}

// registerDomainRoutes memanggil RegisterXxxRoutes tiap domain.
func registerDomainRoutes(r Routes, h Handlers) {
	RegisterSolutionRoutes(r, h.Solution)
	RegisterJWKSRoutes(r, h.JWKS)
	RegisterAuthRoutes(r, h.Auth)
	RegisterUserRoutes(r, h.User)
	RegisterSolutionClientRoutes(r, h.Client)
	RegisterTenantRoutes(r, h.Tenant)
	RegisterOrganizationRoutes(r, h.Org)
	RegisterBenefitRoutes(r, h.Benefit)
}
