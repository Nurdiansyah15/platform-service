package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

// RegisterDashboardRoutes — ringkasan untuk dashboard console (endpoint pembantu, bukan spec).
func RegisterDashboardRoutes(r Routes, h *api.DashboardHandler) {
	r.Authenticated.GET("/admin/summary", middleware.RequirePlatformAdmin(), h.AdminSummary)
}
