package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

const scopeEventsRead = "events:read"

// RegisterWebhookRoutes — Webhooks & Events (spec §3.18).
func RegisterWebhookRoutes(r Routes, h *api.WebhookHandler) {
	r.Authenticated.POST("/solutions/:solutionId/webhooks", middleware.RequirePlatformAdmin(), h.Create)
	r.Authenticated.GET("/events", middleware.RequireService(scopeEventsRead), h.ListEvents)
}
