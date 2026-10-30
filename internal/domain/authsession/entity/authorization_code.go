package entity

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"time"

	"github.com/google/uuid"

	"platform-service/internal/domain/authsession/constant"
	domainerr "platform-service/internal/domain/errors"
)

// AuthorizationCode adalah kode sekali pakai alur SSO console → solution (OAuth2
// authorization code + PKCE). Diterbitkan dari sesi console (ParentSessionID) dan
// ditukar solution menjadi sesi baru pada SolutionID.
type AuthorizationCode struct {
	ID                  string
	CodeHash            string
	UserID              string
	SolutionID          string
	CompanyID           *string
	ParentSessionID     string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	ExpiresAt           time.Time
	UsedAt              *time.Time
	CreatedAt           time.Time
}

type NewAuthorizationCodeParams struct {
	CodeHash            string
	UserID              string
	SolutionID          string
	CompanyID           *string
	ParentSessionID     string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	TTL                 time.Duration
}

func NewAuthorizationCode(p NewAuthorizationCodeParams, now time.Time) (*AuthorizationCode, error) {
	if p.CodeHash == "" || p.UserID == "" || p.SolutionID == "" || p.ParentSessionID == "" ||
		p.RedirectURI == "" || p.TTL <= 0 {
		return nil, domainerr.New(constant.CodeSessionInvalid)
	}
	if p.CodeChallengeMethod != constant.CodeChallengeMethodS256 || !validCodeChallenge(p.CodeChallenge) {
		return nil, domainerr.New(constant.CodeAuthCodeChallengeInvalid)
	}
	now = now.UTC()
	return &AuthorizationCode{
		ID:                  uuid.NewString(),
		CodeHash:            p.CodeHash,
		UserID:              p.UserID,
		SolutionID:          p.SolutionID,
		CompanyID:           p.CompanyID,
		ParentSessionID:     p.ParentSessionID,
		RedirectURI:         p.RedirectURI,
		CodeChallenge:       p.CodeChallenge,
		CodeChallengeMethod: p.CodeChallengeMethod,
		ExpiresAt:           now.Add(p.TTL),
		CreatedAt:           now,
	}, nil
}

// Redeem memeriksa kode bisa ditukar: belum dipakai, belum kedaluwarsa, redirect_uri sama
// persis dengan saat authorize, dan code_verifier cocok dengan code_challenge (S256).
// Semua kegagalan dilaporkan sebagai CodeInvalidGrant. Penandaan "sudah dipakai" disimpan
// repository (MarkUsed) secara compare-and-swap.
func (c *AuthorizationCode) Redeem(codeVerifier, redirectURI string, now time.Time) error {
	invalid := domainerr.New(constant.CodeInvalidGrant)
	if c.UsedAt != nil || !now.Before(c.ExpiresAt) {
		return invalid
	}
	if c.RedirectURI != redirectURI {
		return invalid
	}
	if !VerifyPKCE(c.CodeChallenge, codeVerifier) {
		return invalid
	}
	t := now.UTC()
	c.UsedAt = &t
	return nil
}

// VerifyPKCE — BASE64URL(SHA256(code_verifier)) == code_challenge (RFC 7636 §4.6),
// dibandingkan constant-time.
func VerifyPKCE(challenge, verifier string) bool {
	if !validCodeVerifier(verifier) {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

// validCodeVerifier — RFC 7636 §4.1: 43–128 karakter [A-Za-z0-9-._~].
func validCodeVerifier(v string) bool {
	if len(v) < 43 || len(v) > 128 {
		return false
	}
	for _, r := range v {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9',
			r == '-', r == '.', r == '_', r == '~':
		default:
			return false
		}
	}
	return true
}

// validCodeChallenge — hasil S256 selalu 43 karakter base64url tanpa padding.
func validCodeChallenge(c string) bool {
	if len(c) != 43 {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(c)
	return err == nil
}
