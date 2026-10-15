package entity

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/role/constant"
	"platform-service/internal/domain/role/valueobject"
)

// Role adalah kumpulan permission dalam satu solution (spec §2.6):
//   - system role: tenant_id nil, is_system true — dibuat platform, tidak bisa diubah
//   - global template: tenant_id nil, is_system false — dibuat platform admin
//   - custom tenant role: tenant_id terisi — dikelola tenant pemilik
type Role struct {
	ID          string
	SolutionID  string
	TenantID    *string
	Code        valueobject.RoleCode
	Name        string
	Description *string
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

type NewRoleParams struct {
	SolutionID  string
	TenantID    *string // nil = global
	Code        string
	Name        string
	Description *string
}

func NewRole(p NewRoleParams, now time.Time) (*Role, error) {
	code, err := valueobject.NewRoleCode(p.Code)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(p.Name)
	if name == "" || utf8.RuneCountInString(name) > constant.NameMaxLength {
		return nil, domainerr.New(constant.CodeRoleNameInvalid)
	}
	now = now.UTC()
	return &Role{
		ID: uuid.NewString(), SolutionID: p.SolutionID, TenantID: p.TenantID, Code: code, Name: name,
		Description: trimOptional(p.Description), CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (r *Role) IsGlobal() bool { return r.TenantID == nil }

// IsOwnedBy — custom role milik tenant tersebut.
func (r *Role) IsOwnedBy(tenantID string) bool {
	return r.TenantID != nil && *r.TenantID == tenantID
}

// EnsureEditable — role sistem tidak boleh diubah.
func (r *Role) EnsureEditable() error {
	if r.IsSystem {
		return domainerr.New(constant.CodeRoleSystemImmutable)
	}
	return nil
}

func trimOptional(v *string) *string {
	if v == nil {
		return nil
	}
	t := strings.TrimSpace(*v)
	if t == "" {
		return nil
	}
	return &t
}
