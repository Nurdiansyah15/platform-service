package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

// RegisterTenantRoutes — Tenant & Membership (spec §3.2). Saat ini khusus token
// user; akses token service (users:read / users:write) menyusul di Fase 10.
func RegisterTenantRoutes(r Routes, h *api.TenantHandler) {
	g := r.Authenticated.Group("/tenants", middleware.RequireUser())
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:tenantId", h.Get)
	g.GET("/:tenantId/members", h.ListMembers)
	g.POST("/:tenantId/members", h.AddMember)
}
