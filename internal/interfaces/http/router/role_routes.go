package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

const scopeRolesWrite = "roles:write"

// RegisterRoleRoutes — Roles global (spec §3.11) & custom company (spec §3.12).
func RegisterRoleRoutes(r Routes, h *api.RoleHandler) {
	g := r.Authenticated
	g.GET("/solutions/:solutionId/roles", h.ListGlobal)
	g.GET("/roles/:roleId/permissions", h.ListPermissions)
	g.POST("/solutions/:solutionId/roles", middleware.RequirePlatformAdmin(), h.CreateGlobal)
	g.PUT("/roles/:roleId/permissions", middleware.RequirePlatformAdmin(), h.ReplaceGlobalPermissions)

	company := g.Group("", middleware.RequireUserOrService(scopeRolesWrite))
	company.GET("/companies/:companyId/solutions/:solutionId/roles", h.ListForCompany)
	company.POST("/companies/:companyId/solutions/:solutionId/roles", h.CreateForCompany)
	company.PUT("/companies/:companyId/roles/:roleId/permissions", h.ReplaceCompanyRolePermissions)
}
