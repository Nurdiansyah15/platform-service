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

type CompanyResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
}
