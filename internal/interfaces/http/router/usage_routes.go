package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

const scopeUsageWrite = "usage:write"

// RegisterUsageRoutes — Usage Reporting (spec §3.19). Khusus token service.
func RegisterUsageRoutes(r Routes, h *api.UsageHandler) {
	g := r.Authenticated.Group("/companies/:companyId/solutions/:solutionId/usage", middleware.RequireService(scopeUsageWrite))
	g.POST("", h.ReportDelta)
	g.PUT("/:benefitCode", h.ReportAbsolute)
}
