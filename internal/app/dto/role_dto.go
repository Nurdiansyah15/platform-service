package dto

import "time"

type CreateRoleRequest struct {
	Code        string  `json:"code" binding:"required,max=100"`
	Name        string  `json:"name" binding:"required,max=255"`
	Description *string `json:"description"`
}

type ListRoleQuery struct {
	Search *string `form:"search"`
	PaginationParams
}

type RoleIDParam struct {
	RoleID string `uri:"roleId" binding:"required,uuid"`
}

type TenantRoleParam struct {
	TenantID string `uri:"tenantId" binding:"required,uuid"`
	RoleID   string `uri:"roleId" binding:"required,uuid"`
}

// ReplaceRolePermissionsRequest — replace-all: daftar ini menjadi seluruh permission role.
type ReplaceRolePermissionsRequest struct {
	PermissionCodes []string `json:"permission_codes" binding:"required,dive,required,max=150"`
}

type RoleResponse struct {
	ID          string  `json:"id"`
	SolutionID  string  `json:"solution_id"`
	TenantID    *string `json:"tenant_id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IsSystem    bool    `json:"is_system"`
	// Scope — "system" | "global" | "tenant".
	Scope     string    `json:"scope"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RolePermissionResponse struct {
	ID     string `json:"id"`
	Code   string `json:"code"`
	Name   string `json:"name"`
	Status string `json:"status"`
}
