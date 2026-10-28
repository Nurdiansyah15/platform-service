package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

// RegisterEntitlementRoutes — Effective Entitlements (spec §3.10).
func RegisterEntitlementRoutes(r Routes, h *api.EntitlementHandler) {
	g := r.Authenticated.Group("/companies/:companyId/solutions/:solutionId/entitlements",
		middleware.RequireUserOrService(scopeEntitlementsRead))
	g.GET("", h.List)
	g.GET("/:benefitCode", h.Get)
}
