package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

// Scope service untuk company & branch (spec §2.8).
const (
	scopeOrgRead  = "org:read"
	scopeOrgWrite = "org:write"
)

// RegisterOrganizationRoutes — Company & Branch (spec §3.3). Token user, atau token
// service ber-scope untuk tenant yang subscribe solution-nya.
func RegisterOrganizationRoutes(r Routes, h *api.OrganizationHandler) {
	g := r.Authenticated
	g.GET("/tenants/:tenantId/companies", middleware.RequireUserOrService(scopeOrgRead), h.ListCompanies)
	g.POST("/tenants/:tenantId/companies", middleware.RequireUserOrService(scopeOrgWrite), h.CreateCompany)
	g.GET("/companies/:companyId/branches", middleware.RequireUserOrService(scopeOrgRead), h.ListBranches)
	g.POST("/companies/:companyId/branches", middleware.RequireUserOrService(scopeOrgWrite), h.CreateBranch)
}
