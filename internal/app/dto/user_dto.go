package dto

import "time"

type CreateUserRequest struct {
	Email string `json:"email" binding:"required,max=255"`
	Name  string `json:"name" binding:"required,max=255"`
	// Password opsional: user tanpa password belum bisa login (mis. menunggu undangan).
	Password        *string `json:"password" binding:"omitempty,min=8,max=72"`
	Status          string  `json:"status" binding:"omitempty,oneof=active inactive suspended"`
	IsPlatformAdmin bool    `json:"is_platform_admin"`
}

type UserResponse struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	Name            string    `json:"name"`
	Status          string    `json:"status"`
	IsPlatformAdmin bool      `json:"is_platform_admin"`
	HasPassword     bool      `json:"has_password"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UserIDParam struct {
	UserID string `uri:"userId" binding:"required,uuid"`
}

type ListUserQuery struct {
	Status          *string `form:"status" binding:"omitempty,oneof=active inactive suspended"`
	IsPlatformAdmin *bool   `form:"is_platform_admin"`
	Search          *string `form:"search"`
	PaginationParams
}

// UpdateUserRequest — field nil tidak diubah. password = reset password oleh admin.
type UpdateUserRequest struct {
	Email           *string `json:"email" binding:"omitempty,max=255"`
	Name            *string `json:"name" binding:"omitempty,max=255"`
	Password        *string `json:"password" binding:"omitempty,min=8,max=72"`
	IsPlatformAdmin *bool   `json:"is_platform_admin"`
}

type UpdateUserStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active inactive suspended"`
}

// UpdateMeRequest — user mengubah datanya sendiri. Ganti password wajib menyertakan current_password.
type UpdateMeRequest struct {
	Name            *string `json:"name" binding:"omitempty,max=255"`
	Password        *string `json:"password" binding:"omitempty,min=8,max=72"`
	CurrentPassword *string `json:"current_password" binding:"required_with=Password,omitempty,max=72"`
}

type UserMembershipResponse struct {
	CompanyID     string    `json:"company_id"`
	CompanyName   string    `json:"company_name"`
	CompanySlug   string    `json:"company_slug"`
	CompanyStatus string    `json:"company_status"`
	MemberStatus  string    `json:"member_status"`
	JoinedAt      time.Time `json:"joined_at"`
}

type UserDetailResponse struct {
	UserResponse
	Memberships []UserMembershipResponse `json:"memberships"`
}
