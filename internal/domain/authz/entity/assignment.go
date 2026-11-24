package entity

import (
	"time"

	"github.com/google/uuid"

	"platform-service/internal/domain/authz/valueobject"
)

// RoleAssignment — role diberikan ke user dalam company. Scopes berisi nilai untuk setiap
// scope_type non-company yang dipakai permission role (mis. {"branch": "<uuid>",
// "department": "42"}); permission ber-scope company tidak butuh nilai. Labels = nama
// tampilan opsional nilai scope custom.
type RoleAssignment struct {
	ID        string
	CompanyID string
	UserID    string
	RoleID    string
	Scopes    map[string]string
	Labels    map[string]string
	CreatedAt time.Time

	// Role diisi saat assignment dimuat dari storage (FindByID); kosong untuk assignment baru.
	Role AssignedRole
}

// AssignedRole — ringkasan role dari assignment yang dimuat (role boleh sudah soft delete).
type AssignedRole struct {
	SolutionID string
	Code       string
}

func NewRoleAssignment(companyID, userID, roleID string, scopes, labels map[string]string, now time.Time) *RoleAssignment {
	if scopes == nil {
		scopes = map[string]string{}
	}
	if labels == nil {
		labels = map[string]string{}
	}
	return &RoleAssignment{
		ID: uuid.NewString(), CompanyID: companyID, UserID: userID, RoleID: roleID,
		Scopes: scopes, Labels: labels, CreatedAt: now.UTC(),
	}
}

// Grant adalah satu permission yang dimiliki user lewat satu assignment, pada scope
// role_permissions.scope_type dengan nilai dari assignment (company = company assignment).
type Grant struct {
	RoleCode       string
	PermissionCode string
	Scope          valueobject.Scope
}

// CheckScope adalah scope yang ditanyakan saat authz check. CompanyID = company pemilik
// scope, agar grant level company berlaku untuk semua scope di bawahnya. ParentBranchID diisi
// untuk scope custom ber-parent branch bila branch induknya diketahui, agar grant di branch
// itu ikut berlaku.
type CheckScope struct {
	Scope          valueobject.Scope
	CompanyID      string
	ParentBranchID string
}
