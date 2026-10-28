package entity

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"platform-service/internal/domain/authsession/constant"
	domainerr "platform-service/internal/domain/errors"
)

// AuthSession adalah satu sesi login user pada satu solution (spec §2.7).
type AuthSession struct {
	ID               string
	UserID           string
	SolutionID       string
	CompanyID        *string // company aktif; nil sebelum user memilih company
	RefreshTokenHash string
	UserAgent        *string
	IPAddress        *string
	ExpiresAt        time.Time
	LastUsedAt       *time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
}

type NewAuthSessionParams struct {
	UserID           string
	SolutionID       string
	CompanyID        *string
	RefreshTokenHash string
	UserAgent        string
	IPAddress        string
	TTL              time.Duration
}

func NewAuthSession(p NewAuthSessionParams, now time.Time) (*AuthSession, error) {
	if p.UserID == "" || p.SolutionID == "" || p.RefreshTokenHash == "" || p.TTL <= 0 {
		return nil, domainerr.New(constant.CodeSessionInvalid)
	}
	now = now.UTC()
	return &AuthSession{
		ID:               uuid.NewString(),
		UserID:           p.UserID,
		SolutionID:       p.SolutionID,
		CompanyID:        p.CompanyID,
		RefreshTokenHash: p.RefreshTokenHash,
		UserAgent:        optional(p.UserAgent),
		IPAddress:        optional(p.IPAddress),
		ExpiresAt:        now.Add(p.TTL),
		CreatedAt:        now,
	}, nil
}

// EnsureUsable menolak sesi yang sudah di-revoke atau kedaluwarsa.
func (s *AuthSession) EnsureUsable(now time.Time) error {
	if s.RevokedAt != nil {
		return domainerr.New(constant.CodeSessionRevoked)
	}
	if !now.Before(s.ExpiresAt) {
		return domainerr.New(constant.CodeSessionExpired)
	}
	return nil
}

// Rotate mengganti refresh token (hash) dan memperpanjang masa berlaku (sliding).
// Mengembalikan hash lama untuk compare-and-swap di repository.
func (s *AuthSession) Rotate(newHash string, ttl time.Duration, now time.Time) (previousHash string, err error) {
	if err := s.EnsureUsable(now); err != nil {
		return "", err
	}
	if newHash == "" || ttl <= 0 {
		return "", domainerr.New(constant.CodeSessionInvalid)
	}
	now = now.UTC()
	previousHash = s.RefreshTokenHash
	s.RefreshTokenHash = newHash
	s.ExpiresAt = now.Add(ttl)
	s.LastUsedAt = &now
	return previousHash, nil
}

// SwitchCompany mengganti company aktif sesi. Refresh token ikut di-rotate agar
// token lama (dengan cid lama) tidak bisa ditukar lagi.
func (s *AuthSession) SwitchCompany(companyID, newHash string, ttl time.Duration, now time.Time) (previousHash string, err error) {
	previousHash, err = s.Rotate(newHash, ttl, now)
	if err != nil {
		return "", err
	}
	s.CompanyID = &companyID
	return previousHash, nil
}

// Revoke mengakhiri sesi. Idempotent: sesi yang sudah revoked tidak berubah.
func (s *AuthSession) Revoke(now time.Time) {
	if s.RevokedAt != nil {
		return
	}
	t := now.UTC()
	s.RevokedAt = &t
}

func optional(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
