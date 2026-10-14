package entity

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/tenant/constant"
	"platform-service/internal/domain/tenant/valueobject"
)

type Tenant struct {
	ID        string
	Name      string
	Slug      valueobject.Slug
	Status    valueobject.TenantStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewTenant(name, slug string, now time.Time) (*Tenant, error) {
	n := strings.TrimSpace(name)
	if n == "" || utf8.RuneCountInString(n) > constant.NameMaxLength {
		return nil, domainerr.New(constant.CodeTenantNameInvalid)
	}
	s, err := valueobject.NewSlug(slug)
	if err != nil {
		return nil, err
	}
	now = now.UTC()
	return &Tenant{
		ID:        uuid.NewString(),
		Name:      n,
		Slug:      s,
		Status:    valueobject.TenantStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (t *Tenant) IsActive() bool {
	return t.Status == valueobject.TenantStatusActive
}
