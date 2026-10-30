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
	// CompanyID company aktif; null bila user belum memilih company (diisi mulai Fase 10).
	CompanyID *string `json:"company_id"`
}

type MeResponse struct {
	User    UserResponse    `json:"user"`
	Session SessionResponse `json:"session"`
}

type SessionResponse struct {
	ID        string          `json:"id"`
	Solution  SolutionSummary `json:"solution"`
	CompanyID *string         `json:"company_id"`
	ExpiresAt time.Time       `json:"expires_at"`
	CreatedAt time.Time       `json:"created_at"`
}

type SolutionSummary struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type SwitchCompanyRequest struct {
	CompanyID string `json:"company_id" binding:"required,uuid"`
}

type SelectableCompanyResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
	// IsActive — company ini adalah company aktif token saat ini.
	IsActive bool `json:"is_active"`
}

// AuthorizeRequest — langkah 1 SSO console → solution (OAuth2 authorization code + PKCE).
// Dipanggil halaman /authorize console dengan token sesi console.
type AuthorizeRequest struct {
	// Solution adalah solutions.code tujuan (menjadi aud token hasil tukar kode).
	Solution    string `json:"solution" binding:"required,max=100"`
	RedirectURI string `json:"redirect_uri" binding:"required,max=2000"`
	// CodeChallenge = BASE64URL(SHA256(code_verifier)), dibuat solution.
	CodeChallenge       string `json:"code_challenge" binding:"required,len=43"`
	CodeChallengeMethod string `json:"code_challenge_method" binding:"required,oneof=S256"`
	// State dikembalikan apa adanya ke redirect_uri (proteksi CSRF di sisi solution).
	State string `json:"state" binding:"max=500"`
	// CompanyID company aktif sesi baru. Kosong → company aktif console, atau satu-satunya
	// company yang bisa dipilih; selain itu sesi dibuat tanpa company.
	CompanyID *string `json:"company_id" binding:"omitempty,uuid"`
}

type AuthorizeResponse struct {
	// RedirectTo = redirect_uri + ?code=...&state=...; console langsung mengarahkan browser ke sini.
	RedirectTo string    `json:"redirect_to"`
	Code       string    `json:"code"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// TokenExchangeRequest — langkah 2 SSO: solution menukar code menjadi sesi & token.
type TokenExchangeRequest struct {
	Code         string `json:"code" binding:"required,max=200"`
	CodeVerifier string `json:"code_verifier" binding:"required,min=43,max=128"`
	// RedirectURI harus sama persis dengan saat authorize.
	RedirectURI string `json:"redirect_uri" binding:"required,max=2000"`
}

// AppResponse — satu shortcut di launcher console / menu apps solution.
type AppResponse struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IconURL     *string `json:"icon_url"`
	LaunchURL   string  `json:"launch_url"`
	// IsCurrent — solution token yang sedang dipakai (aud).
	IsCurrent bool `json:"is_current"`
}
