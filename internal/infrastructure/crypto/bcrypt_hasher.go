// Package crypto berisi implementasi primitif kriptografi (hash password, dsb.).
package crypto

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher implements port.PasswordHasher.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher membuat hasher. cost <= 0 → bcrypt.DefaultCost.
// Test boleh memakai bcrypt.MinCost agar cepat.
func NewBcryptHasher(cost int) *BcryptHasher {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

func (h *BcryptHasher) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt: hash password: %w", err)
	}
	return string(b), nil
}

func (h *BcryptHasher) Verify(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
