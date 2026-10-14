package entity

import (
	"strings"
	"time"

	"github.com/google/uuid"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/signingkey/constant"
	"platform-service/internal/domain/signingkey/valueobject"
)

type SigningKey struct {
	ID            string
	KID           string
	Algorithm     string
	PublicKeyPEM  string
	PrivateKeyRef string
	Status        valueobject.KeyStatus
	CreatedAt     time.Time
	RetiredAt     *time.Time
}

// NewSigningKey membuat key active baru dari material yang sudah dibuat key store.
func NewSigningKey(publicKeyPEM, privateKeyRef string, now time.Time) (*SigningKey, error) {
	if strings.TrimSpace(publicKeyPEM) == "" || strings.TrimSpace(privateKeyRef) == "" {
		return nil, domainerr.New(constant.CodeSigningKeyInvalid)
	}
	now = now.UTC()
	return &SigningKey{
		ID: uuid.NewString(),
		// kid diawali tanggal agar mudah dibaca saat audit/rotasi.
		KID:           now.Format("20060102") + "-" + uuid.NewString()[:8],
		Algorithm:     constant.AlgorithmRS256,
		PublicKeyPEM:  publicKeyPEM,
		PrivateKeyRef: privateKeyRef,
		Status:        valueobject.KeyStatusActive,
		CreatedAt:     now,
	}, nil
}

// StartRetiring menghentikan key dari penandatanganan; key tetap dipublikasikan.
func (k *SigningKey) StartRetiring() error {
	if k.Status != valueobject.KeyStatusActive {
		return domainerr.New(constant.CodeSigningKeyTransitionInvalid)
	}
	k.Status = valueobject.KeyStatusRetiring
	return nil
}

// Retire mengeluarkan key dari JWKS.
func (k *SigningKey) Retire(now time.Time) error {
	if k.Status != valueobject.KeyStatusRetiring {
		return domainerr.New(constant.CodeSigningKeyTransitionInvalid)
	}
	t := now.UTC()
	k.Status = valueobject.KeyStatusRetired
	k.RetiredAt = &t
	return nil
}

func (k *SigningKey) IsActive() bool {
	return k.Status == valueobject.KeyStatusActive
}
