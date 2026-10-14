package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

// Scope service untuk user & membership (spec §2.8).
const (
	scopeUsersRead  = "users:read"
	scopeUsersWrite = "users:write"
)

// RegisterTenantRoutes — Tenant & Membership (spec §3.2).
//   - membuat & mendaftar tenant: khusus token user (alur onboarding / pemilihan tenant)
//   - per-tenant: token user, atau token service ber-scope untuk tenant yang subscribe solution-nya
func RegisterTenantRoutes(r Routes, h *api.TenantHandler) {
	user := r.Authenticated.Group("/tenants", middleware.RequireUser())
	user.POST("", h.Create)
	user.GET("", h.List)

	g := r.Authenticated.Group("/tenants")
	g.GET("/:tenantId", middleware.RequireUserOrService(scopeUsersRead), h.Get)
	g.GET("/:tenantId/members", middleware.RequireUserOrService(scopeUsersRead), h.ListMembers)
	g.POST("/:tenantId/members", middleware.RequireUserOrService(scopeUsersWrite), h.AddMember)
}
