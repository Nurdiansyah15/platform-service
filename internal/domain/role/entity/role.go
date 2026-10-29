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
//   - system role: company_id nil, is_system true — dibuat platform, tidak bisa diubah
//   - global template: company_id nil, is_system false — dibuat platform admin
//   - custom company role: company_id terisi — dikelola company pemilik
type Role struct {
	ID          string
	SolutionID  string
	CompanyID   *string
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
	CompanyID   *string // nil = global
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
		ID: uuid.NewString(), SolutionID: p.SolutionID, CompanyID: p.CompanyID, Code: code, Name: name,
		Description: trimOptional(p.Description), CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (r *Role) IsGlobal() bool { return r.CompanyID == nil }

// IsOwnedBy — custom role milik company tersebut.
func (r *Role) IsOwnedBy(companyID string) bool {
	return r.CompanyID != nil && *r.CompanyID == companyID
}

// EnsureEditable — role sistem tidak boleh diubah.
func (r *Role) EnsureEditable() error {
	if r.IsSystem {
		return domainerr.New(constant.CodeRoleSystemImmutable)
	}
	return nil
}

// Update mengganti nama dan/atau deskripsi; nil = tidak diubah. Role sistem tidak boleh diubah.
func (r *Role) Update(name, description *string, now time.Time) error {
	if err := r.EnsureEditable(); err != nil {
		return err
	}
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" || utf8.RuneCountInString(n) > constant.NameMaxLength {
			return domainerr.New(constant.CodeRoleNameInvalid)
		}
		r.Name = n
	}
	if description != nil {
		r.Description = trimOptional(description)
	}
	r.UpdatedAt = now.UTC()
	return nil
}

// Delete menandai role terhapus (soft delete). Role sistem tidak boleh dihapus.
func (r *Role) Delete(now time.Time) error {
	if err := r.EnsureEditable(); err != nil {
		return err
	}
	t := now.UTC()
	r.DeletedAt = &t
	r.UpdatedAt = t
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
