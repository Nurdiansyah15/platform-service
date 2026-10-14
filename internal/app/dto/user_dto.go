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
