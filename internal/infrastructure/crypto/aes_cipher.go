package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

// AESCipher implements port.SecretCipher dengan AES-256-GCM. Ciphertext = base64(nonce || sealed).
type AESCipher struct {
	aead cipher.AEAD
}

// NewAESCipher menerima kunci base64 32 byte.
func NewAESCipher(base64Key string) (*AESCipher, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("aes: kunci bukan base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("aes: kunci harus 32 byte, didapat %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	return &AESCipher{aead: aead}, nil
}

func (c *AESCipher) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("aes: nonce: %w", err)
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (c *AESCipher) Decrypt(ciphertext string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("aes: ciphertext bukan base64: %w", err)
	}
	n := c.aead.NonceSize()
	if len(raw) < n {
		return "", errors.New("aes: ciphertext terlalu pendek")
	}
	plain, err := c.aead.Open(nil, raw[:n], raw[n:], nil)
	if err != nil {
		return "", fmt.Errorf("aes: dekripsi gagal: %w", err)
	}
	return string(plain), nil
}
