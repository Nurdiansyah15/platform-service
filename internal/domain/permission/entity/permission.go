package entity

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/permission/constant"
	"platform-service/internal/domain/permission/valueobject"
)

type Permission struct {
	ID          string
	SolutionID  string
	Code        valueobject.PermissionCode
	Name        string
	Description *string
	Status      valueobject.PermissionStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Definition adalah satu permission dalam katalog yang dikirim solution.
type Definition struct {
	Code        valueobject.PermissionCode
	Name        string
	Description *string
}

func NewDefinition(code, name string, description *string) (Definition, error) {
	c, err := valueobject.NewPermissionCode(code)
	if err != nil {
		return Definition{}, err
	}
	n := strings.TrimSpace(name)
	if n == "" || utf8.RuneCountInString(n) > constant.NameMaxLength {
		return Definition{}, domainerr.New(constant.CodePermissionNameInvalid)
	}
	return Definition{Code: c, Name: n, Description: trimOptional(description)}, nil
}

func NewPermission(solutionID string, d Definition, now time.Time) *Permission {
	now = now.UTC()
	return &Permission{
		ID: uuid.NewString(), SolutionID: solutionID, Code: d.Code, Name: d.Name, Description: d.Description,
		Status: valueobject.StatusActive, CreatedAt: now, UpdatedAt: now,
	}
}

func (p *Permission) IsActive() bool { return p.Status == valueobject.StatusActive }

// Apply menyesuaikan permission dengan definisi katalog. Mengembalikan apakah ada
// perubahan dan apakah permission diaktifkan kembali dari deprecated.
func (p *Permission) Apply(d Definition, now time.Time) (changed, reactivated bool) {
	if p.Status == valueobject.StatusDeprecated {
		p.Status = valueobject.StatusActive
		changed, reactivated = true, true
	}
	if p.Name != d.Name || !equalOptional(p.Description, d.Description) {
		p.Name, p.Description = d.Name, d.Description
		changed = true
	}
	if changed {
		p.UpdatedAt = now.UTC()
	}
	return changed, reactivated
}

// Deprecate menandai permission tidak lagi ada di katalog. Mengembalikan false bila sudah deprecated.
func (p *Permission) Deprecate(now time.Time) bool {
	if p.Status == valueobject.StatusDeprecated {
		return false
	}
	p.Status = valueobject.StatusDeprecated
	p.UpdatedAt = now.UTC()
	return true
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

func equalOptional(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
