package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

// RegisterBenefitRoutes — Master Benefits (spec §3.7).
func RegisterBenefitRoutes(r Routes, h *api.BenefitHandler) {
	r.Authenticated.GET("/solutions/:solutionId/benefits", h.List)
	r.Authenticated.GET("/benefits/:benefitId", h.Get)
	r.Authenticated.POST("/solutions/:solutionId/benefits",
		middleware.RequirePlatformAdmin(), r.Deployment.LicenseManaged(), h.Create)
}
