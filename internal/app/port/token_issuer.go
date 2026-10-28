package port

import (
	"context"
	"time"
)

// UserTokenClaims — isi token user (spec §2.8). Permission & entitlement cidak dimasukkan.
type UserTokenClaims struct {
	UserID       string // sub
	SolutionCode string // aud
	CompanyID    string // cid, boleh kosong sebelum pilih company
	SessionID    string // sid
}

// ServiceTokenClaims — isi token service hasil client credentials.
type ServiceTokenClaims struct {
	ClientID   string // sub
	SolutionID string // sol
	Scopes     []string
}

type IssuedToken struct {
	Token     string
	ExpiresAt time.Time
}

// TokenIssuer menandatangani access token dengan signing key active.
type TokenIssuer interface {
	IssueUserToken(ctx context.Context, c UserTokenClaims) (IssuedToken, error)
	IssueServiceToken(ctx context.Context, c ServiceTokenClaims) (IssuedToken, error)
}
