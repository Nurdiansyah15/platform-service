package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

const scopeCatalogSync = "catalog:sync"

// RegisterPermissionRoutes — Permissions & Catalog Sync (spec §3.13, §3.20).
func RegisterPermissionRoutes(r Routes, h *api.PermissionHandler) {
	r.Authenticated.GET("/solutions/:solutionId/permissions", h.List)
	r.Authenticated.POST("/solutions/:solutionId/permissions", middleware.RequirePlatformAdmin(), h.Create)
	r.Authenticated.GET("/permissions/:permissionId", h.Get)
	r.Authenticated.PATCH("/permissions/:permissionId", middleware.RequirePlatformAdmin(), h.Update)
	r.Authenticated.GET("/solutions/:solutionId/permission-groups", h.ListGroups)
	r.Authenticated.GET("/solutions/:solutionId/scope-types", h.ListScopeTypes)
	// User: hanya platform admin (dicek usecase). Service: catalog:sync milik solution tersebut.
	r.Authenticated.PUT("/solutions/:solutionId/permissions/sync", middleware.RequireUserOrService(scopeCatalogSync), h.Sync)
}
