package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

// RegisterOrganizationRoutes — Company & Branch (spec §3.3). Saat ini khusus token
// user; akses token service (org:read / org:write) menyusul di Fase 10.
func RegisterOrganizationRoutes(r Routes, h *api.OrganizationHandler) {
	g := r.Authenticated.Group("", middleware.RequireUser())
	g.GET("/tenants/:tenantId/companies", h.ListCompanies)
	g.POST("/tenants/:tenantId/companies", h.CreateCompany)
	g.GET("/companies/:companyId/branches", h.ListBranches)
	g.POST("/companies/:companyId/branches", h.CreateBranch)
}
