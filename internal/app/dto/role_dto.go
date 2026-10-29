package dto

import "time"

type CreateRoleRequest struct {
	Code        string  `json:"code" binding:"required,max=100"`
	Name        string  `json:"name" binding:"required,max=255"`
	Description *string `json:"description"`
}

type ListRoleQuery struct {
	Search *string `form:"search"`
	// Kind — global (termasuk sistem) | custom (milik company). Kosong = semua.
	Kind *string `form:"kind" binding:"omitempty,oneof=global custom"`
	PaginationParams
}

type RoleIDParam struct {
	RoleID string `uri:"roleId" binding:"required,uuid"`
}

type CompanyRoleParam struct {
	CompanyID string `uri:"companyId" binding:"required,uuid"`
	RoleID    string `uri:"roleId" binding:"required,uuid"`
}

// ReplaceRolePermissionsRequest — replace-all: daftar ini menjadi seluruh permission role.
type ReplaceRolePermissionsRequest struct {
	PermissionCodes []string `json:"permission_codes" binding:"required,dive,required,max=150"`
}

type RoleResponse struct {
	ID          string  `json:"id"`
	SolutionID  string  `json:"solution_id"`
	CompanyID   *string `json:"company_id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IsSystem    bool    `json:"is_system"`
	// Scope — "system" | "global" | "company".
	Scope     string    `json:"scope"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// PermissionCount & MemberCount — jumlah permission dan user pemegang role. Dalam konteks
	// company, MemberCount hanya menghitung pemegang di company tersebut.
	PermissionCount int `json:"permission_count"`
	MemberCount     int `json:"member_count"`
}

// UpdateRoleRequest — field nil tidak diubah; description "" = dikosongkan.
type UpdateRoleRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=255"`
	Description *string `json:"description" binding:"omitempty,max=1000"`
}

// RoleHolderResponse — satu assignment role pada user di company.
type RoleHolderResponse struct {
	AssignmentID string    `json:"assignment_id"`
	UserID       string    `json:"user_id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	ScopeType    string    `json:"scope_type"`
	ScopeID      string    `json:"scope_id"`
	ScopeName    string    `json:"scope_name"`
	CreatedAt    time.Time `json:"created_at"`
}

type RolePermissionResponse struct {
	ID     string `json:"id"`
	Code   string `json:"code"`
	Name   string `json:"name"`
	Status string `json:"status"`
}
