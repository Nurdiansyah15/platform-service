package entity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/signingkey/constant"
	"platform-service/internal/domain/signingkey/entity"
	"platform-service/internal/domain/signingkey/valueobject"
	"platform-service/internal/testhelper"
)

func TestNewSigningKey(t *testing.T) {
	now := time.Date(2026, 10, 14, 7, 0, 0, 0, time.UTC)
	k, err := entity.NewSigningKey("-----BEGIN PUBLIC KEY-----", "mem:abc", now)

	require.NoError(t, err)
	assert.Regexp(t, `^20261014-[0-9a-f]{8}$`, k.KID)
	assert.Equal(t, "RS256", k.Algorithm)
	assert.True(t, k.IsActive())
	assert.True(t, k.Status.IsPublished())
}

func TestNewSigningKey_Invalid(t *testing.T) {
	_, err := entity.NewSigningKey("", "mem:abc", time.Now())
	testhelper.AssertDomainError(t, err, constant.CodeSigningKeyInvalid)

	_, err = entity.NewSigningKey("pem", " ", time.Now())
	testhelper.AssertDomainError(t, err, constant.CodeSigningKeyInvalid)
}

func TestSigningKey_Lifecycle(t *testing.T) {
	k, err := entity.NewSigningKey("pem", "mem:abc", time.Now())
	require.NoError(t, err)

	testhelper.AssertDomainError(t, k.Retire(time.Now()), constant.CodeSigningKeyTransitionInvalid)

	require.NoError(t, k.StartRetiring())
	assert.Equal(t, valueobject.KeyStatusRetiring, k.Status)
	assert.True(t, k.Status.IsPublished())
	testhelper.AssertDomainError(t, k.StartRetiring(), constant.CodeSigningKeyTransitionInvalid)

	require.NoError(t, k.Retire(time.Now()))
	assert.Equal(t, valueobject.KeyStatusRetired, k.Status)
	assert.False(t, k.Status.IsPublished())
	require.NotNil(t, k.RetiredAt)
}
