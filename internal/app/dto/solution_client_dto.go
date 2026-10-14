package dto

import (
	"net/http"
	"time"
)

type CreateSolutionClientRequest struct {
	Name   string   `json:"name" binding:"required,max=255"`
	Scopes []string `json:"scopes" binding:"omitempty,dive,max=50"`
}

type SolutionClientIDParam struct {
	ClientID string `uri:"clientId" binding:"required,uuid"`
}

type SolutionClientResponse struct {
	ID         string     `json:"id"`
	SolutionID string     `json:"solution_id"`
	Name       string     `json:"name"`
	ClientID   string     `json:"client_id"`
	Scopes     []string   `json:"scopes"`
	Status     string     `json:"status"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
}

// SolutionClientWithSecretResponse — dikembalikan saat create / rotate; secret
// hanya bisa dilihat sekali ini.
type SolutionClientWithSecretResponse struct {
	SolutionClientResponse
	ClientSecret string `json:"client_secret"`
}

// ── OAuth2 client credentials (RFC 6749 §4.4) ────────────────────────────────

// ClientCredentialsRequest dibaca dari form (atau JSON). client_id/client_secret
// boleh juga datang dari header HTTP Basic.
type ClientCredentialsRequest struct {
	GrantType    string `form:"grant_type" json:"grant_type"`
	ClientID     string `form:"client_id" json:"client_id"`
	ClientSecret string `form:"client_secret" json:"client_secret"`
	Scope        string `form:"scope" json:"scope"`
	// UsedBasicAuth diisi handler; menentukan header WWW-Authenticate saat 401.
	UsedBasicAuth bool `form:"-" json:"-"`
}

// OAuthTokenResponse — format standar RFC 6749 §5.1, tanpa envelope.
type OAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	Scope       string `json:"scope"`
}

// OAuthError adalah error RFC 6749 §5.2. Dikirim apa adanya (tanpa envelope).
type OAuthError struct {
	Status      int    `json:"-"`
	Code        string `json:"error"`
	Description string `json:"error_description,omitempty"`
	BasicAuth   bool   `json:"-"`
}

func (e *OAuthError) Error() string { return "oauth: " + e.Code + ": " + e.Description }

func NewOAuthInvalidRequest(desc string) *OAuthError {
	return &OAuthError{Status: http.StatusBadRequest, Code: "invalid_request", Description: desc}
}

func NewOAuthInvalidClient(basicAuth bool) *OAuthError {
	return &OAuthError{Status: http.StatusUnauthorized, Code: "invalid_client", Description: "client authentication failed", BasicAuth: basicAuth}
}

func NewOAuthUnsupportedGrantType() *OAuthError {
	return &OAuthError{Status: http.StatusBadRequest, Code: "unsupported_grant_type", Description: "only client_credentials is supported"}
}

func NewOAuthInvalidScope() *OAuthError {
	return &OAuthError{Status: http.StatusBadRequest, Code: "invalid_scope", Description: "requested scope is unknown or exceeds the client's scopes"}
}
