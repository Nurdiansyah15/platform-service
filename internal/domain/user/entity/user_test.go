package entity_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/user/constant"
	"platform-service/internal/domain/user/entity"
	"platform-service/internal/testhelper"
)

func TestNewUser_Valid(t *testing.T) {
	u, err := entity.NewUser(entity.NewUserParams{Email: "  Budi@Example.COM ", Name: " Budi "}, time.Now())

	require.NoError(t, err)
	assert.Equal(t, "budi@example.com", u.Email.Value())
	assert.Equal(t, "Budi", u.Name)
	assert.True(t, u.IsActive())
	assert.False(t, u.HasPassword())
	assert.False(t, u.IsPlatformAdmin)
}

func TestNewUser_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		params entity.NewUserParams
		want   domainerr.Code
	}{
		{"email kosong", entity.NewUserParams{Email: "", Name: "X"}, constant.CodeUserEmailInvalid},
		{"email tanpa @", entity.NewUserParams{Email: "budi.example.com", Name: "X"}, constant.CodeUserEmailInvalid},
		{"email display name", entity.NewUserParams{Email: "Budi <budi@example.com>", Name: "X"}, constant.CodeUserEmailInvalid},
		{"email tanpa domain titik", entity.NewUserParams{Email: "budi@localhost", Name: "X"}, constant.CodeUserEmailInvalid},
		{"name kosong", entity.NewUserParams{Email: "budi@example.com", Name: " "}, constant.CodeUserNameInvalid},
		{"status tidak dikenal", entity.NewUserParams{Email: "budi@example.com", Name: "X", Status: "banned"}, constant.CodeUserStatusInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entity.NewUser(tt.params, time.Now())
			testhelper.AssertDomainError(t, err, tt.want)
		})
	}
}

func TestValidatePassword(t *testing.T) {
	assert.NoError(t, entity.ValidatePassword("Secret123"))
	testhelper.AssertDomainError(t, entity.ValidatePassword("short"), constant.CodeUserPasswordInvalid)
	testhelper.AssertDomainError(t, entity.ValidatePassword(strings.Repeat("a", 73)), constant.CodeUserPasswordInvalid)
}
