package entity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/authsession/constant"
	"platform-service/internal/domain/authsession/entity"
	"platform-service/internal/testhelper"
)

var t0 = time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC)

func newSession(t *testing.T) *entity.AuthSession {
	t.Helper()
	s, err := entity.NewAuthSession(entity.NewAuthSessionParams{
		UserID: "u", SolutionID: "sol", RefreshTokenHash: "h1", UserAgent: " curl ", TTL: time.Hour,
	}, t0)
	require.NoError(t, err)
	return s
}

func TestNewAuthSession(t *testing.T) {
	s := newSession(t)
	assert.NotEmpty(t, s.ID)
	assert.Equal(t, t0.Add(time.Hour), s.ExpiresAt)
	require.NotNil(t, s.UserAgent)
	assert.Equal(t, "curl", *s.UserAgent)
	assert.Nil(t, s.IPAddress)
	assert.NoError(t, s.EnsureUsable(t0))

	_, err := entity.NewAuthSession(entity.NewAuthSessionParams{UserID: "u", SolutionID: "sol", TTL: time.Hour}, t0)
	testhelper.AssertDomainError(t, err, constant.CodeSessionInvalid)
}

func TestAuthSession_Expiry(t *testing.T) {
	s := newSession(t)
	assert.NoError(t, s.EnsureUsable(t0.Add(59*time.Minute)))
	testhelper.AssertDomainError(t, s.EnsureUsable(t0.Add(time.Hour)), constant.CodeSessionExpired)
}

func TestAuthSession_RotateSlidesExpiry(t *testing.T) {
	s := newSession(t)
	later := t0.Add(30 * time.Minute)

	prev, err := s.Rotate("h2", time.Hour, later)
	require.NoError(t, err)
	assert.Equal(t, "h1", prev)
	assert.Equal(t, "h2", s.RefreshTokenHash)
	assert.Equal(t, later.Add(time.Hour), s.ExpiresAt)
	require.NotNil(t, s.LastUsedAt)
}

func TestAuthSession_RevokeBlocksRotate(t *testing.T) {
	s := newSession(t)
	s.Revoke(t0)
	first := *s.RevokedAt
	s.Revoke(t0.Add(time.Minute))
	assert.Equal(t, first, *s.RevokedAt, "revoke idempotent")

	_, err := s.Rotate("h2", time.Hour, t0)
	testhelper.AssertDomainError(t, err, constant.CodeSessionRevoked)
	assert.Equal(t, "h1", s.RefreshTokenHash)
}

func TestAuthSession_SwitchTenant(t *testing.T) {
	s := newSession(t)
	prev, err := s.SwitchTenant("tenant-1", "h2", time.Hour, t0)
	require.NoError(t, err)
	assert.Equal(t, "h1", prev)
	require.NotNil(t, s.TenantID)
	assert.Equal(t, "tenant-1", *s.TenantID)
	assert.Equal(t, "h2", s.RefreshTokenHash)

	s.Revoke(t0)
	_, err = s.SwitchTenant("tenant-2", "h3", time.Hour, t0)
	testhelper.AssertDomainError(t, err, constant.CodeSessionRevoked)
}
