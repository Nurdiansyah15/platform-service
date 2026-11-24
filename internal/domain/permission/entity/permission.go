package entity

import (
	"slices"
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
	// Source — sync (dikelola sync katalog, read-only di platform) atau manual.
	Source valueobject.PermissionSource
	// GroupID — permission group (nil = tanpa grup; UI mengelompokkan by prefix code).
	GroupID *string
	// ScopeTypes — scope type tempat permission berlaku (terurut). Kosong = semua scope.
	ScopeTypes []string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Definition adalah satu permission dalam katalog yang dikirim solution.
type Definition struct {
	Code        valueobject.PermissionCode
	Name        string
	Description *string
	// GroupCode — code group yang dirujuk ("" = tanpa grup). GroupID diisi usecase
	// setelah group tersinkron.
	GroupCode string
	GroupID   *string
	// Scopes — scope type tempat permission berlaku (unik, terurut). Kosong = semua scope.
	Scopes []string
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

// WithCatalogRefs memvalidasi format referensi group & scope. Keberadaan group / scope type
// dicek terhadap katalog lewat service.ValidateCatalogRefs.
func (d Definition) WithCatalogRefs(groupCode *string, scopes []string) (Definition, error) {
	if g := trimOptional(groupCode); g != nil {
		gc, err := valueobject.NewGroupCode(*g)
		if err != nil {
			return Definition{}, err
		}
		d.GroupCode = gc.Value()
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(scopes))
	for _, raw := range scopes {
		s := strings.TrimSpace(raw)
		if !valueobject.IsBuiltinScopeType(s) {
			if _, err := valueobject.NewScopeTypeCode(s); err != nil {
				return Definition{}, err
			}
		}
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	slices.Sort(out)
	d.Scopes = out
	return d, nil
}

// NewPermission membuat permission hasil sync katalog.
func NewPermission(solutionID string, d Definition, now time.Time) *Permission {
	now = now.UTC()
	return &Permission{
		ID: uuid.NewString(), SolutionID: solutionID, Code: d.Code, Name: d.Name, Description: d.Description,
		Status: valueobject.StatusActive, Source: valueobject.SourceSync, GroupID: d.GroupID,
		ScopeTypes: scopesOrEmpty(d.Scopes), CreatedAt: now, UpdatedAt: now,
	}
}

// NewManualPermission membuat permission yang dikelola platform admin (tidak disentuh sync).
func NewManualPermission(solutionID string, d Definition, now time.Time) *Permission {
	p := NewPermission(solutionID, d, now)
	p.Source = valueobject.SourceManual
	return p
}

func (p *Permission) IsManual() bool { return p.Source == valueobject.SourceManual }

// EnsureManual — permission hasil sync dikelola kode solution dan read-only di platform.
func (p *Permission) EnsureManual() error {
	if !p.IsManual() {
		return domainerr.New(constant.CodePermissionSyncManaged)
	}
	return nil
}

// Edit mengubah atribut permission manual (code & status tidak berubah). Mengembalikan
// apakah ada perubahan.
func (p *Permission) Edit(d Definition, now time.Time) bool {
	if p.Name == d.Name && equalOptional(p.Description, d.Description) &&
		equalOptional(p.GroupID, d.GroupID) && slices.Equal(scopesOrEmpty(p.ScopeTypes), scopesOrEmpty(d.Scopes)) {
		return false
	}
	p.Name, p.Description = d.Name, d.Description
	p.GroupID, p.ScopeTypes = d.GroupID, scopesOrEmpty(d.Scopes)
	p.UpdatedAt = now.UTC()
	return true
}

// SetStatus mengaktifkan / men-deprecate permission. Mengembalikan apakah status berubah.
func (p *Permission) SetStatus(s valueobject.PermissionStatus, now time.Time) bool {
	if p.Status == s {
		return false
	}
	p.Status = s
	p.UpdatedAt = now.UTC()
	return true
}

func (p *Permission) IsActive() bool { return p.Status == valueobject.StatusActive }

// Apply menyesuaikan permission dengan definisi katalog. Mengembalikan apakah ada
// perubahan dan apakah permission diaktifkan kembali dari deprecated.
func (p *Permission) Apply(d Definition, now time.Time) (changed, reactivated bool) {
	if p.Status == valueobject.StatusDeprecated {
		p.Status = valueobject.StatusActive
		changed, reactivated = true, true
	}
	if p.Name != d.Name || !equalOptional(p.Description, d.Description) ||
		!equalOptional(p.GroupID, d.GroupID) || !slices.Equal(scopesOrEmpty(p.ScopeTypes), scopesOrEmpty(d.Scopes)) {
		p.Name, p.Description = d.Name, d.Description
		p.GroupID, p.ScopeTypes = d.GroupID, scopesOrEmpty(d.Scopes)
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

// AppliesTo — permission berlaku di scope type tersebut (ScopeTypes kosong = semua scope).
func (p *Permission) AppliesTo(scopeType string) bool {
	return len(p.ScopeTypes) == 0 || slices.Contains(p.ScopeTypes, scopeType)
}

func scopesOrEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
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
