package dto

import "time"

type TenantUserParam struct {
	TenantID string `uri:"tenantId" binding:"required,uuid"`
	UserID   string `uri:"userId" binding:"required,uuid"`
}

type CreateRoleAssignmentRequest struct {
	RoleID string `json:"role_id" binding:"required,uuid"`
	// ScopeType — tenant (default) | company | branch.
	ScopeType string `json:"scope_type" binding:"omitempty,oneof=tenant company branch"`
	// ScopeID — wajib untuk company / branch; untuk tenant boleh kosong (= tenant itu sendiri).
	ScopeID string `json:"scope_id" binding:"omitempty,uuid"`
}

type RoleAssignmentResponse struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	UserID     string    `json:"user_id"`
	RoleID     string    `json:"role_id"`
	RoleCode   string    `json:"role_code"`
	SolutionID string    `json:"solution_id"`
	ScopeType  string    `json:"scope_type"`
	ScopeID    string    `json:"scope_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// AuthzCheckRequest — token service wajib mengisi tenant_id & user_id; token user
// mengecek dirinya sendiri pada tenant aktif token (tid).
type AuthzCheckRequest struct {
	TenantID   string `json:"tenant_id" binding:"omitempty,uuid"`
	UserID     string `json:"user_id" binding:"omitempty,uuid"`
	Permission string `json:"permission" binding:"required,max=150"`
	ScopeType  string `json:"scope_type" binding:"omitempty,oneof=tenant company branch"`
	ScopeID    string `json:"scope_id" binding:"omitempty,uuid"`
}

type AuthzCheckBatchRequest struct {
	TenantID string           `json:"tenant_id" binding:"omitempty,uuid"`
	UserID   string           `json:"user_id" binding:"omitempty,uuid"`
	Checks   []AuthzCheckItem `json:"checks" binding:"required,min=1,max=100,dive"`
}

type AuthzCheckItem struct {
	Permission string `json:"permission" binding:"required,max=150"`
	ScopeType  string `json:"scope_type" binding:"omitempty,oneof=tenant company branch"`
	ScopeID    string `json:"scope_id" binding:"omitempty,uuid"`
}

type AuthzCheckResult struct {
	Permission string `json:"permission"`
	ScopeType  string `json:"scope_type"`
	ScopeID    string `json:"scope_id"`
	Allowed    bool   `json:"allowed"`
	// GrantedBy — code role yang memberi izin.
	GrantedBy []string `json:"granted_by"`
}

type AuthzCheckBatchResponse struct {
	TenantID string             `json:"tenant_id"`
	UserID   string             `json:"user_id"`
	Results  []AuthzCheckResult `json:"results"`
}
