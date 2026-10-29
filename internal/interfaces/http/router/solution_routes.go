package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

// RegisterSolutionRoutes — Solution Catalog (spec §3.4).
func RegisterSolutionRoutes(r Routes, h *api.SolutionHandler) {
	g := r.Authenticated.Group("/solutions")
	g.GET("", h.List)
	g.GET("/:solutionId", h.Get)
	g.POST("", middleware.RequirePlatformAdmin(), r.Deployment.LicenseManaged(), h.Create)
	g.PATCH("/:solutionId", middleware.RequirePlatformAdmin(), r.Deployment.LicenseManaged(), h.Update)
}
