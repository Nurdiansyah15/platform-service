package dto

import "time"

type CompanyUserParam struct {
	CompanyID string `uri:"companyId" binding:"required,uuid"`
	UserID    string `uri:"userId" binding:"required,uuid"`
}

type CreateRoleAssignmentRequest struct {
	RoleID string `json:"role_id" binding:"required,uuid"`
	// ScopeValues — nilai untuk setiap scope_type non-company yang dipakai permission role,
	// mis. {"branch": "<uuid branch>", "department": "42"}. Role yang hanya berisi permission
	// ber-scope company tidak butuh nilai. Kunci di luar scope role ditolak.
	ScopeValues map[string]string `json:"scope_values"`
	// ScopeLabels — nama tampilan opsional untuk nilai scope custom, mis. {"department": "Keuangan"}.
	ScopeLabels map[string]string `json:"scope_labels"`
}

// AssignmentScope — satu nilai scope assignment beserta nama tampilannya.
type AssignmentScope struct {
	Type string `json:"type"`
	// TypeName — Company / Branch / nama scope type dari katalog solution.
	TypeName string `json:"type_name"`
	Value    string `json:"value"`
	// Name — nama branch, label scope custom, atau value bila tanpa label.
	Name string `json:"name"`
}

type RoleAssignmentResponse struct {
	ID          string            `json:"id"`
	CompanyID   string            `json:"company_id"`
	UserID      string            `json:"user_id"`
	RoleID      string            `json:"role_id"`
	RoleCode    string            `json:"role_code"`
	SolutionID  string            `json:"solution_id"`
	ScopeValues map[string]string `json:"scope_values"`
	ScopeLabels map[string]string `json:"scope_labels"`
	CreatedAt   time.Time         `json:"created_at"`
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
	ID           string            `json:"id"`
	CompanyID    string            `json:"company_id"`
	UserID       string            `json:"user_id"`
	RoleID       string            `json:"role_id"`
	RoleCode     string            `json:"role_code"`
	RoleName     string            `json:"role_name"`
	RoleIsGlobal bool              `json:"role_is_global"`
	SolutionID   string            `json:"solution_id"`
	SolutionCode string            `json:"solution_code"`
	ScopeValues  map[string]string `json:"scope_values"`
	ScopeLabels  map[string]string `json:"scope_labels"`
	// Scopes — nilai scope untuk ditampilkan (urut type).
	Scopes []AssignmentScope `json:"scopes"`
	// RoleScopeTypes — scope_type unik permission role saat ini (company/branch/custom). Menentukan
	// level role.assign yang dibutuhkan untuk mencabut: ada selain branch → level company.
	RoleScopeTypes []string  `json:"role_scope_types"`
	CreatedAt      time.Time `json:"created_at"`
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
	Name   string                   `json:"name"`
	Group  *PermissionGroupRef      `json:"group"`
	Grants []EffectiveGrantResponse `json:"grants"`
}
