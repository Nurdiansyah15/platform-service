package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

// RegisterAddonRoutes — Add-ons & Add-on Benefits (spec §3.4, §3.6).
func RegisterAddonRoutes(r Routes, h *api.AddonHandler) {
	r.Authenticated.GET("/solutions/:solutionId/addons", h.List)
	r.Authenticated.GET("/addons/:addonId/benefits", h.ListBenefits)

	admin := r.Authenticated.Group("", middleware.RequirePlatformAdmin(), r.Deployment.LicenseManaged())
	admin.POST("/solutions/:solutionId/addons", h.Create)
	admin.PUT("/addons/:addonId/benefits", h.ReplaceBenefits)
}
