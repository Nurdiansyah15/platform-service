package entity

import (
	"time"

	"github.com/google/uuid"

	"platform-service/internal/domain/authz/valueobject"
)

// RoleAssignment — role diberikan ke user dalam company pada scope tertentu.
type RoleAssignment struct {
	ID        string
	CompanyID string
	UserID    string
	RoleID    string
	Scope     valueobject.Scope
	// ScopeLabel — nama tampilan scope custom (data milik solution; platform tidak tahu namanya).
	ScopeLabel *string
	// ScopeParentID — branch induk untuk scope custom ber-parent branch (opsional).
	ScopeParentID *string
	CreatedAt     time.Time

	// Role diisi saat assignment dimuat dari storage (FindByID); kosong untuk assignment baru.
	Role AssignedRole
}

// AssignedRole — ringkasan role dari assignment yang dimuat (role boleh sudah soft delete).
type AssignedRole struct {
	SolutionID string
	Code       string
}

// NewRoleAssignment membuat assignment pada scope yang sudah divalidasi. Label & parent hanya
// disimpan untuk scope custom.
func NewRoleAssignment(companyID, userID, roleID string, scope CheckScope, label *string, now time.Time) *RoleAssignment {
	a := &RoleAssignment{
		ID: uuid.NewString(), CompanyID: companyID, UserID: userID, RoleID: roleID, Scope: scope.Scope, CreatedAt: now.UTC(),
	}
	if !scope.Scope.Type.IsBuiltin() {
		a.ScopeLabel = label
		if scope.ParentBranchID != "" {
			p := scope.ParentBranchID
			a.ScopeParentID = &p
		}
	}
	return a
}

// CheckScope merekonstruksi scope check dari assignment tersimpan.
func (a *RoleAssignment) CheckScope() CheckScope {
	c := CheckScope{Scope: a.Scope, CompanyID: a.CompanyID}
	if a.ScopeParentID != nil {
		c.ParentBranchID = *a.ScopeParentID
	}
	return c
}

// Grant adalah satu permission yang dimiliki user lewat satu assignment.
type Grant struct {
	RoleCode       string
	PermissionCode string
	// PermissionScopes — scope type tempat permission berlaku (kosong = semua scope).
	PermissionScopes []string
	Scope            valueobject.Scope
}

// CheckScope adalah scope yang ditanyakan saat authz check (atau scope assignment yang
// sudah divalidasi). CompanyID = company pemilik scope, agar assignment level company
// berlaku untuk semua scope di bawahnya. ParentBranchID diisi untuk scope custom ber-parent
// branch bila branch induknya diketahui, agar assignment di branch itu ikut berlaku.
type CheckScope struct {
	Scope          valueobject.Scope
	CompanyID      string
	ParentBranchID string
}
