package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

// RegisterSolutionClientRoutes — Solution Clients (spec §3.17) & OAuth token (spec §3.16).
func RegisterSolutionClientRoutes(r Routes, h *api.SolutionClientHandler) {
	r.V1.POST("/oauth/token", h.Token)

	admin := r.Authenticated.Group("", middleware.RequirePlatformAdmin())
	admin.GET("/solutions/:solutionId/clients", h.List)
	admin.POST("/solutions/:solutionId/clients", h.Create)
	admin.GET("/solution-clients/:clientId", h.Get)
	admin.PATCH("/solution-clients/:clientId", h.Update)
	admin.POST("/solution-clients/:clientId/rotate-secret", h.RotateSecret)
	admin.POST("/solution-clients/:clientId/revoke", h.Revoke)
}
