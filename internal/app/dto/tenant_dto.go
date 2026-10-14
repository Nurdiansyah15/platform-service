package dto

import "time"

type CreateTenantRequest struct {
	Name string `json:"name" binding:"required,max=255"`
	Slug string `json:"slug" binding:"required,max=100"`
}

type ListTenantQuery struct {
	Status *string `form:"status" binding:"omitempty,oneof=active suspended closed"`
	Search *string `form:"search"`
	PaginationParams
}

type TenantIDParam struct {
	TenantID string `uri:"tenantId" binding:"required,uuid"`
}

type TenantResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AddTenantMemberRequest — isi salah satu: user_id atau email (user harus sudah terdaftar).
type AddTenantMemberRequest struct {
	UserID *string `json:"user_id" binding:"omitempty,uuid"`
	Email  *string `json:"email" binding:"omitempty,max=255"`
	Status string  `json:"status" binding:"omitempty,oneof=active invited"`
}

type ListTenantMemberQuery struct {
	Status *string `form:"status" binding:"omitempty,oneof=invited active inactive"`
	Search *string `form:"search"`
	PaginationParams
}

type TenantMemberResponse struct {
	TenantID string    `json:"tenant_id"`
	UserID   string    `json:"user_id"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Status   string    `json:"status"`
	JoinedAt time.Time `json:"joined_at"`
}
