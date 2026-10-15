package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/infrastructure/crypto"
)

const testKey = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=" // 32 byte

func TestAESCipher(t *testing.T) {
	c, err := crypto.NewAESCipher(testKey)
	require.NoError(t, err)

	a, err := c.Encrypt("whsec_rahasia")
	require.NoError(t, err)
	b, err := c.Encrypt("whsec_rahasia")
	require.NoError(t, err)
	assert.NotEqual(t, a, b, "nonce acak")
	assert.NotContains(t, a, "rahasia")

	plain, err := c.Decrypt(a)
	require.NoError(t, err)
	assert.Equal(t, "whsec_rahasia", plain)

	other, err := crypto.NewAESCipher("ZmVkY2JhOTg3NjU0MzIxMGZlZGNiYTk4NzY1NDMyMTA=")
	require.NoError(t, err)
	_, err = other.Decrypt(a)
	assert.Error(t, err, "kunci lain gagal")

	_, err = crypto.NewAESCipher("cGVuZGVr")
	assert.Error(t, err, "kunci bukan 32 byte")
}
