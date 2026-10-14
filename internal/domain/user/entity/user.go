package entity

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/user/constant"
	"platform-service/internal/domain/user/valueobject"
)

type User struct {
	ID              string
	Email           valueobject.Email
	Name            string
	PasswordHash    *string // nil = user belum punya password (mis. diundang)
	Status          valueobject.UserStatus
	IsPlatformAdmin bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type NewUserParams struct {
	Email           string
	Name            string
	PasswordHash    *string // sudah di-hash oleh usecase; validasi panjang via ValidatePassword
	Status          string  // kosong → active
	IsPlatformAdmin bool
}

func NewUser(p NewUserParams, now time.Time) (*User, error) {
	email, err := valueobject.NewEmail(p.Email)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(p.Name)
	if name == "" || utf8.RuneCountInString(name) > constant.NameMaxLength {
		return nil, domainerr.New(constant.CodeUserNameInvalid)
	}
	status := valueobject.UserStatusActive
	if p.Status != "" {
		if status, err = valueobject.NewUserStatus(p.Status); err != nil {
			return nil, err
		}
	}
	now = now.UTC()
	return &User{
		ID:              uuid.NewString(),
		Email:           email,
		Name:            name,
		PasswordHash:    p.PasswordHash,
		Status:          status,
		IsPlatformAdmin: p.IsPlatformAdmin,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// ValidatePassword memeriksa aturan password mentah sebelum di-hash.
func ValidatePassword(raw string) error {
	if utf8.RuneCountInString(raw) < constant.PasswordMinLength || len(raw) > constant.PasswordMaxLength {
		return domainerr.New(constant.CodeUserPasswordInvalid)
	}
	return nil
}

func (u *User) IsActive() bool {
	return u.Status == valueobject.UserStatusActive
}

func (u *User) HasPassword() bool {
	return u.PasswordHash != nil && *u.PasswordHash != ""
}

// GrantPlatformAdmin menjadikan user administrator platform.
func (u *User) GrantPlatformAdmin(now time.Time) {
	u.IsPlatformAdmin = true
	u.UpdatedAt = now.UTC()
}
