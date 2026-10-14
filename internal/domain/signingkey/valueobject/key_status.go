package valueobject

import (
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/signingkey/constant"
)

// KeyStatus — siklus rotasi: active → retiring → retired.
//   - active:   dipakai menandatangani, dipublikasikan di JWKS
//   - retiring: tidak lagi menandatangani, masih di JWKS sampai token lama kedaluwarsa
//   - retired:  dikeluarkan dari JWKS; token yang ditandatanganinya ditolak
type KeyStatus string

const (
	KeyStatusActive   KeyStatus = "active"
	KeyStatusRetiring KeyStatus = "retiring"
	KeyStatusRetired  KeyStatus = "retired"
)

func NewKeyStatus(raw string) (KeyStatus, error) {
	switch s := KeyStatus(raw); s {
	case KeyStatusActive, KeyStatusRetiring, KeyStatusRetired:
		return s, nil
	default:
		return "", domainerr.New(constant.CodeSigningKeyStatusInvalid)
	}
}

func (s KeyStatus) String() string { return string(s) }

// IsPublished — key yang muncul di JWKS dan boleh dipakai memverifikasi token.
func (s KeyStatus) IsPublished() bool {
	return s == KeyStatusActive || s == KeyStatusRetiring
}
