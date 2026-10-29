package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

const scopeAuthzCheck = "authz:check"

// RegisterAuthzRoutes — User Role Assignments (spec §3.14) & Authorization Check (spec §3.15).
func RegisterAuthzRoutes(r Routes, h *api.AuthzHandler) {
	member := r.Authenticated.Group("/companies/:companyId/users/:userId")
	member.GET("/role-assignments", middleware.RequireUserOrService(scopeRolesWrite), h.ListAssignments)
	member.POST("/role-assignments", middleware.RequireUserOrService(scopeRolesWrite), h.AssignRole)
	member.GET("/effective-permissions", middleware.RequireUserOrService(scopeAuthzCheck), h.EffectivePermissions)
	r.Authenticated.DELETE("/role-assignments/:assignmentId", middleware.RequireUserOrService(scopeRolesWrite), h.RevokeAssignment)

	check := r.Authenticated.Group("/authz", middleware.RequireUserOrService(scopeAuthzCheck))
	check.POST("/check", h.Check)
	check.POST("/check-batch", h.CheckBatch)
}
