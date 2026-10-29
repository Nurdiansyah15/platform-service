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

// Update mengganti nama dan/atau email; nil = tidak diubah.
func (u *User) Update(name, email *string, now time.Time) error {
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" || utf8.RuneCountInString(n) > constant.NameMaxLength {
			return domainerr.New(constant.CodeUserNameInvalid)
		}
		u.Name = n
	}
	if email != nil {
		e, err := valueobject.NewEmail(*email)
		if err != nil {
			return err
		}
		u.Email = e
	}
	u.UpdatedAt = now.UTC()
	return nil
}

// SetPassword mengganti hash password (sudah divalidasi & di-hash oleh usecase).
func (u *User) SetPassword(hash string, now time.Time) {
	u.PasswordHash = &hash
	u.UpdatedAt = now.UTC()
}

// ChangeStatus mengganti status user. Mengembalikan true bila status benar-benar berubah.
func (u *User) ChangeStatus(raw string, now time.Time) (bool, error) {
	st, err := valueobject.NewUserStatus(raw)
	if err != nil {
		return false, err
	}
	if st == u.Status {
		return false, nil
	}
	u.Status = st
	u.UpdatedAt = now.UTC()
	return true, nil
}

// SetPlatformAdmin memberi / mencabut flag administrator platform.
func (u *User) SetPlatformAdmin(admin bool, now time.Time) {
	u.IsPlatformAdmin = admin
	u.UpdatedAt = now.UTC()
}

// GrantPlatformAdmin menjadikan user administrator platform.
func (u *User) GrantPlatformAdmin(now time.Time) {
	u.IsPlatformAdmin = true
	u.UpdatedAt = now.UTC()
}
