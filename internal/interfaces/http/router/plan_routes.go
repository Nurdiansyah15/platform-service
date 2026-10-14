package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

// RegisterPlanRoutes — Plans & Plan Benefits (spec §3.4, §3.5).
func RegisterPlanRoutes(r Routes, h *api.PlanHandler) {
	r.Authenticated.GET("/solutions/:solutionId/plans", h.List)
	r.Authenticated.GET("/plans/:planId/benefits", h.ListBenefits)

	admin := r.Authenticated.Group("", middleware.RequirePlatformAdmin(), r.Deployment.LicenseManaged())
	admin.POST("/solutions/:solutionId/plans", h.Create)
	admin.PUT("/plans/:planId/benefits", h.ReplaceBenefits)
}
