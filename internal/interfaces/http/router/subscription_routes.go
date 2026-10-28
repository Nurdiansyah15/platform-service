package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

// Scope service untuk membaca subscription & entitlement (spec §2.8).
const scopeEntitlementsRead = "entitlements:read"

// RegisterSubscriptionRoutes — Subscription & Subscription Add-ons (spec §3.8, §3.9).
func RegisterSubscriptionRoutes(r Routes, h *api.SubscriptionHandler) {
	read := r.Authenticated.Group("", middleware.RequireUserOrService(scopeEntitlementsRead))
	read.GET("/companies/:companyId/subscriptions", h.List)
	read.GET("/subscriptions/:subscriptionId/addons", h.ListAddons)

	admin := r.Authenticated.Group("", middleware.RequirePlatformAdmin(), r.Deployment.LicenseManaged())
	admin.POST("/companies/:companyId/subscriptions", h.Create)
	admin.POST("/subscriptions/:subscriptionId/change-plan", h.ChangePlan)
	admin.POST("/subscriptions/:subscriptionId/cancel", h.Cancel)
	admin.POST("/subscriptions/:subscriptionId/addons", h.AddAddon)
}
