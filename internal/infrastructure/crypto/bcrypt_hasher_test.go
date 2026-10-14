package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"platform-service/internal/infrastructure/crypto"
)

func TestBcryptHasher(t *testing.T) {
	h := crypto.NewBcryptHasher(bcrypt.MinCost)
	hash, err := h.Hash("Secret123")
	require.NoError(t, err)

	assert.NotEqual(t, "Secret123", hash)
	assert.True(t, h.Verify(hash, "Secret123"))
	assert.False(t, h.Verify(hash, "secret123"))
	assert.False(t, h.Verify("bukan-hash", "Secret123"))
}
