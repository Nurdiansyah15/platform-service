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

// Scope service untuk branch (spec §2.8).
const (
	scopeOrgRead  = "org:read"
	scopeOrgWrite = "org:write"
)

// RegisterOrganizationRoutes — Company (entitas tertinggi), Membership, dan Branch.
//   - membuat & mendaftar company: khusus token user (alur onboarding / pemilihan company)
//   - per-company: token user, atau token service ber-scope untuk company yang subscribe solution-nya
func RegisterOrganizationRoutes(r Routes, h *api.OrganizationHandler) {
	user := r.Authenticated.Group("/companies", middleware.RequireUser())
	user.POST("", h.Create)
	user.GET("", h.List)

	g := r.Authenticated.Group("/companies")
	g.GET("/:companyId", middleware.RequireUserOrService(scopeUsersRead), h.Get)
	g.PATCH("/:companyId", middleware.RequireUserOrService(scopeOrgWrite), h.Update)
	g.GET("/:companyId/members", middleware.RequireUserOrService(scopeUsersRead), h.ListMembers)
	g.POST("/:companyId/members", middleware.RequireUserOrService(scopeUsersWrite), h.AddMember)
	g.GET("/:companyId/members/:userId", middleware.RequireUserOrService(scopeUsersRead), h.GetMember)
	g.PATCH("/:companyId/members/:userId", middleware.RequireUserOrService(scopeUsersWrite), h.UpdateMember)
	g.DELETE("/:companyId/members/:userId", middleware.RequireUserOrService(scopeUsersWrite), h.RemoveMember)
	g.GET("/:companyId/summary", middleware.RequireUserOrService(scopeUsersRead), h.Summary)
	g.GET("/:companyId/member-candidates", middleware.RequireUserOrService(scopeUsersWrite), h.MemberCandidates)
	g.GET("/:companyId/branches", middleware.RequireUserOrService(scopeOrgRead), h.ListBranches)
	g.POST("/:companyId/branches", middleware.RequireUserOrService(scopeOrgWrite), h.CreateBranch)

	branch := r.Authenticated.Group("/branches", middleware.RequireUserOrService(scopeOrgWrite))
	branch.PATCH("/:branchId", h.UpdateBranch)
	branch.DELETE("/:branchId", h.DeleteBranch)
}
