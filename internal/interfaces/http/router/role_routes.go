package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

const scopeRolesWrite = "roles:write"

// RegisterRoleRoutes — Roles global (spec §3.11) & custom tenant (spec §3.12).
func RegisterRoleRoutes(r Routes, h *api.RoleHandler) {
	g := r.Authenticated
	g.GET("/solutions/:solutionId/roles", h.ListGlobal)
	g.GET("/roles/:roleId/permissions", h.ListPermissions)
	g.POST("/solutions/:solutionId/roles", middleware.RequirePlatformAdmin(), h.CreateGlobal)
	g.PUT("/roles/:roleId/permissions", middleware.RequirePlatformAdmin(), h.ReplaceGlobalPermissions)

	tenant := g.Group("", middleware.RequireUserOrService(scopeRolesWrite))
	tenant.GET("/tenants/:tenantId/solutions/:solutionId/roles", h.ListForTenant)
	tenant.POST("/tenants/:tenantId/solutions/:solutionId/roles", h.CreateForTenant)
	tenant.PUT("/tenants/:tenantId/roles/:roleId/permissions", h.ReplaceTenantRolePermissions)
}
