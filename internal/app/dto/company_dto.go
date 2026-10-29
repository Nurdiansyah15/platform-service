package dto

import "time"

type CreateCompanyRequest struct {
	Name string `json:"name" binding:"required,max=255"`
	Slug string `json:"slug" binding:"required,max=100"`
}

type ListCompanyQuery struct {
	Status *string `form:"status" binding:"omitempty,oneof=active suspended closed"`
	Search *string `form:"search"`
	PaginationParams
}

// UpdateCompanyRequest — field nil tidak diubah. status hanya boleh diubah platform admin.
type UpdateCompanyRequest struct {
	Name   *string `json:"name" binding:"omitempty,max=255"`
	Slug   *string `json:"slug" binding:"omitempty,max=100"`
	Status *string `json:"status" binding:"omitempty,oneof=active suspended closed"`
}

type CompanyResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// MemberCount — jumlah member active (hanya diisi di list).
	MemberCount int `json:"member_count"`
}

// AddCompanyMemberRequest — isi salah satu: user_id atau email (user harus sudah terdaftar).
type AddCompanyMemberRequest struct {
	UserID *string `json:"user_id" binding:"omitempty,uuid"`
	Email  *string `json:"email" binding:"omitempty,max=255"`
	Status string  `json:"status" binding:"omitempty,oneof=active invited"`
}

type ListCompanyMemberQuery struct {
	Status *string `form:"status" binding:"omitempty,oneof=invited active inactive"`
	Search *string `form:"search"`
	PaginationParams
}

type CompanyMemberResponse struct {
	CompanyID string    `json:"company_id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	JoinedAt  time.Time `json:"joined_at"`
	// RoleCount — jumlah role assignment member di company (semua solution; hanya diisi di list).
	RoleCount int `json:"role_count"`
}

type CompanyMemberParam struct {
	CompanyID string `uri:"companyId" binding:"required,uuid"`
	UserID    string `uri:"userId" binding:"required,uuid"`
}

// UpdateCompanyMemberRequest — status inactive = member dikeluarkan (role assignment ikut dihapus).
type UpdateCompanyMemberRequest struct {
	Status string `json:"status" binding:"required,oneof=invited active inactive"`
}

// MemberCandidateQuery — pencarian user terdaftar untuk ditambahkan sebagai member.
type MemberCandidateQuery struct {
	Search string `form:"search" binding:"required,min=3,max=100"`
}

// MemberCandidateResponse — data minimal user (tanpa status / flag admin).
type MemberCandidateResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CompanySummaryResponse struct {
	Members struct {
		Active   int `json:"active"`
		Invited  int `json:"invited"`
		Inactive int `json:"inactive"`
	} `json:"members"`
	Branches        int `json:"branches"`
	CustomRoles     int `json:"custom_roles"`
	RoleAssignments int `json:"role_assignments"`
}
