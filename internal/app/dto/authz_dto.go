package dto

import "time"

type CompanyUserParam struct {
	CompanyID string `uri:"companyId" binding:"required,uuid"`
	UserID    string `uri:"userId" binding:"required,uuid"`
}

type CreateRoleAssignmentRequest struct {
	RoleID string `json:"role_id" binding:"required,uuid"`
	// ScopeType — company (default) | branch | scope type custom solution role (GET /solutions/{id}/scope-types).
	ScopeType string `json:"scope_type" binding:"omitempty,max=30"`
	// ScopeID — company boleh kosong (= company itu sendiri); branch = UUID branch;
	// scope custom = id milik solution (string bebas, wajib).
	ScopeID string `json:"scope_id" binding:"omitempty,max=150"`
	// ParentID — scope custom ber-parent branch: UUID branch induk (opsional). Membuat
	// pemegang role.assign di branch itu boleh mengelola assignment ini.
	ParentID string `json:"parent_id" binding:"omitempty,uuid"`
	// ScopeLabel — nama tampilan scope custom (opsional; diabaikan untuk company/branch).
	ScopeLabel *string `json:"scope_label" binding:"omitempty,max=255"`
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
	ScopeLabel *string   `json:"scope_label"`
	ParentID   *string   `json:"parent_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// AuthzCheckRequest — token service wajib mengisi company_id & user_id; token user
// mengecek dirinya sendiri pada company aktif token (cid).
type AuthzCheckRequest struct {
	CompanyID  string `json:"company_id" binding:"omitempty,uuid"`
	UserID     string `json:"user_id" binding:"omitempty,uuid"`
	Permission string `json:"permission" binding:"required,max=150"`
	ScopeType  string `json:"scope_type" binding:"omitempty,max=30"`
	ScopeID    string `json:"scope_id" binding:"omitempty,max=150"`
	// ParentID — scope custom ber-parent branch: UUID branch induk (opsional), agar
	// assignment di branch itu ikut berlaku.
	ParentID string `json:"parent_id" binding:"omitempty,uuid"`
}

type AuthzCheckBatchRequest struct {
	CompanyID string           `json:"company_id" binding:"omitempty,uuid"`
	UserID    string           `json:"user_id" binding:"omitempty,uuid"`
	Checks    []AuthzCheckItem `json:"checks" binding:"required,min=1,max=100,dive"`
}

type AuthzCheckItem struct {
	Permission string `json:"permission" binding:"required,max=150"`
	ScopeType  string `json:"scope_type" binding:"omitempty,max=30"`
	ScopeID    string `json:"scope_id" binding:"omitempty,max=150"`
	ParentID   string `json:"parent_id" binding:"omitempty,uuid"`
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

type RoleAssignmentIDParam struct {
	AssignmentID string `uri:"assignmentId" binding:"required,uuid"`
}

// SolutionFilterQuery — solution_id untuk endpoint per-user. Token service selalu
// memakai solution client (nilai ini diabaikan).
type SolutionFilterQuery struct {
	SolutionID *string `form:"solution_id" binding:"omitempty,uuid"`
}

// RoleAssignmentDetailResponse — assignment beserta info role & scope untuk ditampilkan.
type RoleAssignmentDetailResponse struct {
	ID           string `json:"id"`
	CompanyID    string `json:"company_id"`
	UserID       string `json:"user_id"`
	RoleID       string `json:"role_id"`
	RoleCode     string `json:"role_code"`
	RoleName     string `json:"role_name"`
	RoleIsGlobal bool   `json:"role_is_global"`
	SolutionID   string `json:"solution_id"`
	SolutionCode string `json:"solution_code"`
	ScopeType    string `json:"scope_type"`
	// ScopeTypeName — nama tampilan scope type (Company / Branch / nama dari katalog solution).
	ScopeTypeName string    `json:"scope_type_name"`
	ScopeID       string    `json:"scope_id"`
	ScopeName     string    `json:"scope_name"`
	ScopeLabel    *string   `json:"scope_label"`
	ParentID      *string   `json:"parent_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type EffectiveGrantResponse struct {
	RoleCode  string `json:"role_code"`
	ScopeType string `json:"scope_type"`
	ScopeID   string `json:"scope_id"`
}

// EffectivePermissionResponse — satu permission yang dimiliki user beserta asal grant-nya.
type EffectivePermissionResponse struct {
	Permission string `json:"permission"`
	// Name & Group — info tampilan dari katalog (Name = code bila tidak ditemukan).
	Name       string                   `json:"name"`
	Group      *PermissionGroupRef      `json:"group"`
	ScopeTypes []string                 `json:"scope_types"`
	Grants     []EffectiveGrantResponse `json:"grants"`
}
