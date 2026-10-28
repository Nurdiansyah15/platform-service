package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

const scopeAuthzCheck = "authz:check"

// RegisterAuthzRoutes — User Role Assignments (spec §3.14) & Authorization Check (spec §3.15).
func RegisterAuthzRoutes(r Routes, h *api.AuthzHandler) {
	r.Authenticated.POST("/companies/:companyId/users/:userId/role-assignments",
		middleware.RequireUserOrService(scopeRolesWrite), h.AssignRole)

	check := r.Authenticated.Group("/authz", middleware.RequireUserOrService(scopeAuthzCheck))
	check.POST("/check", h.Check)
	check.POST("/check-batch", h.CheckBatch)
}
