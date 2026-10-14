package valueobject

import (
	"net/mail"
	"strings"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/user/constant"
)

// Email ter-normalisasi: di-trim dan lowercase, sehingga unik secara case-insensitive.
type Email struct {
	value string
}

func NewEmail(raw string) (Email, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" || len(v) > constant.EmailMaxLength {
		return Email{}, domainerr.New(constant.CodeUserEmailInvalid)
	}
	addr, err := mail.ParseAddress(v)
	// Tolak bentuk "Nama <email>" — hanya alamat polos yang diterima.
	if err != nil || addr.Address != v || !strings.Contains(v[strings.LastIndex(v, "@"):], ".") {
		return Email{}, domainerr.New(constant.CodeUserEmailInvalid)
	}
	return Email{value: v}, nil
}

func (e Email) Value() string { return e.value }
