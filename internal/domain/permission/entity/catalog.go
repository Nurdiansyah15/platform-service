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

// Group — pengelompokan permission untuk tampilan (mis. "Cuti", "Payroll"). Didefinisikan
// solution lewat sync katalog; tidak pernah dihapus, yang hilang dari katalog jadi deprecated.
type Group struct {
	ID          string
	SolutionID  string
	Code        valueobject.GroupCode
	Name        string
	Description *string
	SortOrder   int
	Status      valueobject.PermissionStatus
	// Source — sync (dikelola sync katalog, read-only di platform) atau manual.
	Source    valueobject.PermissionSource
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GroupDefinition — satu group dalam katalog. SortOrder = posisi di katalog.
type GroupDefinition struct {
	Code        valueobject.GroupCode
	Name        string
	Description *string
	SortOrder   int
}

func NewGroupDefinition(code, name string, description *string, sortOrder int) (GroupDefinition, error) {
	c, err := valueobject.NewGroupCode(code)
	if err != nil {
		return GroupDefinition{}, err
	}
	n, ok := validName(name)
	if !ok {
		return GroupDefinition{}, domainerr.New(constant.CodeGroupNameInvalid)
	}
	return GroupDefinition{Code: c, Name: n, Description: trimOptional(description), SortOrder: sortOrder}, nil
}

func NewGroup(solutionID string, d GroupDefinition, now time.Time) *Group {
	now = now.UTC()
	return &Group{
		ID: uuid.NewString(), SolutionID: solutionID, Code: d.Code, Name: d.Name, Description: d.Description,
		SortOrder: d.SortOrder, Status: valueobject.StatusActive, Source: valueobject.SourceSync,
		CreatedAt: now, UpdatedAt: now,
	}
}

// NewManualGroup membuat group yang dikelola platform admin (tidak disentuh sync).
func NewManualGroup(solutionID string, d GroupDefinition, now time.Time) *Group {
	g := NewGroup(solutionID, d, now)
	g.Source = valueobject.SourceManual
	return g
}

func (g *Group) IsActive() bool { return g.Status == valueobject.StatusActive }

func (g *Group) IsManual() bool { return g.Source == valueobject.SourceManual }

// EnsureManual — group hasil sync dikelola kode solution dan read-only di platform.
func (g *Group) EnsureManual() error {
	if !g.IsManual() {
		return domainerr.New(constant.CodeGroupSyncManaged)
	}
	return nil
}

// Edit mengubah nama, deskripsi, dan urutan group manual (code & status tetap).
func (g *Group) Edit(d GroupDefinition, now time.Time) bool {
	if g.Name == d.Name && equalOptional(g.Description, d.Description) && g.SortOrder == d.SortOrder {
		return false
	}
	g.Name, g.Description, g.SortOrder = d.Name, d.Description, d.SortOrder
	g.UpdatedAt = now.UTC()
	return true
}

// SetStatus mengaktifkan / men-deprecate group. Mengembalikan apakah status berubah.
func (g *Group) SetStatus(s valueobject.PermissionStatus, now time.Time) bool {
	if g.Status == s {
		return false
	}
	g.Status = s
	g.UpdatedAt = now.UTC()
	return true
}

func (g *Group) Apply(d GroupDefinition, now time.Time) (changed, reactivated bool) {
	if g.Status == valueobject.StatusDeprecated {
		g.Status = valueobject.StatusActive
		changed, reactivated = true, true
	}
	if g.Name != d.Name || !equalOptional(g.Description, d.Description) || g.SortOrder != d.SortOrder {
		g.Name, g.Description, g.SortOrder = d.Name, d.Description, d.SortOrder
		changed = true
	}
	if changed {
		g.UpdatedAt = now.UTC()
	}
	return changed, reactivated
}

func (g *Group) Deprecate(now time.Time) bool {
	if g.Status == valueobject.StatusDeprecated {
		return false
	}
	g.Status = valueobject.StatusDeprecated
	g.UpdatedAt = now.UTC()
	return true
}

// ScopeType — scope type custom milik solution (mis. "department"). company & branch
// built-in dan tidak disimpan sebagai ScopeType. Parent menentukan hierarki: grant di
// parent (company selalu; branch bila Parent = branch) menutupi scope ini.
type ScopeType struct {
	ID          string
	SolutionID  string
	Code        valueobject.ScopeTypeCode
	Name        string
	Description *string
	Parent      valueobject.ScopeParent
	SortOrder   int
	Status      valueobject.PermissionStatus
	// Source — sync (dikelola sync katalog, read-only di platform) atau manual.
	Source    valueobject.PermissionSource
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ScopeTypeDefinition struct {
	Code        valueobject.ScopeTypeCode
	Name        string
	Description *string
	Parent      valueobject.ScopeParent
	SortOrder   int
}

func NewScopeTypeDefinition(code, name string, description *string, parent string, sortOrder int) (ScopeTypeDefinition, error) {
	c, err := valueobject.NewScopeTypeCode(code)
	if err != nil {
		return ScopeTypeDefinition{}, err
	}
	n, ok := validName(name)
	if !ok {
		return ScopeTypeDefinition{}, domainerr.New(constant.CodeScopeTypeNameInvalid)
	}
	p, err := valueobject.NewScopeParent(parent)
	if err != nil {
		return ScopeTypeDefinition{}, err
	}
	return ScopeTypeDefinition{Code: c, Name: n, Description: trimOptional(description), Parent: p, SortOrder: sortOrder}, nil
}

func NewScopeType(solutionID string, d ScopeTypeDefinition, now time.Time) *ScopeType {
	now = now.UTC()
	return &ScopeType{
		ID: uuid.NewString(), SolutionID: solutionID, Code: d.Code, Name: d.Name, Description: d.Description,
		Parent: d.Parent, SortOrder: d.SortOrder, Status: valueobject.StatusActive, Source: valueobject.SourceSync,
		CreatedAt: now, UpdatedAt: now,
	}
}

// NewManualScopeType membuat scope type yang dikelola platform admin (tidak disentuh sync).
func NewManualScopeType(solutionID string, d ScopeTypeDefinition, now time.Time) *ScopeType {
	s := NewScopeType(solutionID, d, now)
	s.Source = valueobject.SourceManual
	return s
}

func (s *ScopeType) IsActive() bool { return s.Status == valueobject.StatusActive }

func (s *ScopeType) IsManual() bool { return s.Source == valueobject.SourceManual }

// EnsureManual — scope type hasil sync dikelola kode solution dan read-only di platform.
func (s *ScopeType) EnsureManual() error {
	if !s.IsManual() {
		return domainerr.New(constant.CodeScopeTypeSyncManaged)
	}
	return nil
}

// Edit mengubah nama, deskripsi, parent, dan urutan scope type manual (code & status tetap).
func (s *ScopeType) Edit(d ScopeTypeDefinition, now time.Time) bool {
	if s.Name == d.Name && equalOptional(s.Description, d.Description) && s.Parent == d.Parent && s.SortOrder == d.SortOrder {
		return false
	}
	s.Name, s.Description, s.Parent, s.SortOrder = d.Name, d.Description, d.Parent, d.SortOrder
	s.UpdatedAt = now.UTC()
	return true
}

// SetStatus mengaktifkan / men-deprecate scope type. Mengembalikan apakah status berubah.
func (s *ScopeType) SetStatus(st valueobject.PermissionStatus, now time.Time) bool {
	if s.Status == st {
		return false
	}
	s.Status = st
	s.UpdatedAt = now.UTC()
	return true
}

func (s *ScopeType) Apply(d ScopeTypeDefinition, now time.Time) (changed, reactivated bool) {
	if s.Status == valueobject.StatusDeprecated {
		s.Status = valueobject.StatusActive
		changed, reactivated = true, true
	}
	if s.Name != d.Name || !equalOptional(s.Description, d.Description) || s.Parent != d.Parent || s.SortOrder != d.SortOrder {
		s.Name, s.Description, s.Parent, s.SortOrder = d.Name, d.Description, d.Parent, d.SortOrder
		changed = true
	}
	if changed {
		s.UpdatedAt = now.UTC()
	}
	return changed, reactivated
}

func (s *ScopeType) Deprecate(now time.Time) bool {
	if s.Status == valueobject.StatusDeprecated {
		return false
	}
	s.Status = valueobject.StatusDeprecated
	s.UpdatedAt = now.UTC()
	return true
}

func validName(raw string) (string, bool) {
	n := strings.TrimSpace(raw)
	return n, n != "" && utf8.RuneCountInString(n) <= constant.NameMaxLength
}
