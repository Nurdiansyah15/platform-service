package dto

import "time"

type LoginRequest struct {
	Email    string `json:"email" binding:"required,max=255"`
	Password string `json:"password" binding:"required,max=72"`
	// Solution adalah solutions.code tujuan login (menjadi aud token).
	Solution string `json:"solution" binding:"required,max=100"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required,max=200"`
}

// ClientInfo dicatat di auth_sessions untuk daftar sesi aktif.
type ClientInfo struct {
	UserAgent string
	IPAddress string
}

type AuthTokenResponse struct {
	AccessToken           string    `json:"access_token"`
	TokenType             string    `json:"token_type"`
	ExpiresIn             int64     `json:"expires_in"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
	SessionID             string    `json:"session_id"`
	// TenantID tenant aktif; null bila user belum memilih tenant (diisi mulai Fase 10).
	TenantID *string `json:"tenant_id"`
}

type MeResponse struct {
	User    UserResponse    `json:"user"`
	Session SessionResponse `json:"session"`
}

type SessionResponse struct {
	ID        string          `json:"id"`
	Solution  SolutionSummary `json:"solution"`
	TenantID  *string         `json:"tenant_id"`
	ExpiresAt time.Time       `json:"expires_at"`
	CreatedAt time.Time       `json:"created_at"`
}

type SolutionSummary struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type SwitchTenantRequest struct {
	TenantID string `json:"tenant_id" binding:"required,uuid"`
}

type SelectableTenantResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
	// IsActive — tenant ini adalah tenant aktif token saat ini.
	IsActive bool `json:"is_active"`
}
