package dto

import "time"

type CompanyUserParam struct {
	CompanyID string `uri:"companyId" binding:"required,uuid"`
	UserID    string `uri:"userId" binding:"required,uuid"`
}

type CreateRoleAssignmentRequest struct {
	RoleID string `json:"role_id" binding:"required,uuid"`
	// ScopeType — company (default) | branch.
	ScopeType string `json:"scope_type" binding:"omitempty,oneof=company branch"`
	// ScopeID — wajib untuk company / branch; untuk company boleh kosong (= company itu sendiri).
	ScopeID string `json:"scope_id" binding:"omitempty,uuid"`
}

type RoleAssignmentResponse struct {
	ID         string    `json:"id"`
	CompanyID  string    `json:"company_id"`
	UserID     string    `json:"user_id"`
	RoleID     string    `json:"role_id"`
	RoleCode   string    `json:"role_code"`
	SolutionID string    `json:"solution_id"`
	ScopeType  string    `json:"scope_type"`
	ScopeID    string    `json:"scope_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// AuthzCheckRequest — token service wajib mengisi company_id & user_id; token user
// mengecek dirinya sendiri pada company aktif token (cid).
type AuthzCheckRequest struct {
	CompanyID  string `json:"company_id" binding:"omitempty,uuid"`
	UserID     string `json:"user_id" binding:"omitempty,uuid"`
	Permission string `json:"permission" binding:"required,max=150"`
	ScopeType  string `json:"scope_type" binding:"omitempty,oneof=company branch"`
	ScopeID    string `json:"scope_id" binding:"omitempty,uuid"`
}

type AuthzCheckBatchRequest struct {
	CompanyID string           `json:"company_id" binding:"omitempty,uuid"`
	UserID    string           `json:"user_id" binding:"omitempty,uuid"`
	Checks    []AuthzCheckItem `json:"checks" binding:"required,min=1,max=100,dive"`
}

type AuthzCheckItem struct {
	Permission string `json:"permission" binding:"required,max=150"`
	ScopeType  string `json:"scope_type" binding:"omitempty,oneof=company branch"`
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
	CompanyID string             `json:"company_id"`
	UserID    string             `json:"user_id"`
	Results   []AuthzCheckResult `json:"results"`
}
