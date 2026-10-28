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
	CreatedAt time.Time
}

func NewRoleAssignment(companyID, userID, roleID string, scope valueobject.Scope, now time.Time) *RoleAssignment {
	return &RoleAssignment{
		ID: uuid.NewString(), CompanyID: companyID, UserID: userID, RoleID: roleID, Scope: scope, CreatedAt: now.UTC(),
	}
}

// Grant adalah satu permission yang dimiliki user lewat satu assignment.
type Grant struct {
	RoleCode       string
	PermissionCode string
	Scope          valueobject.Scope
}

// CheckScope adalah scope yang ditanyakan saat authz check. BranchCompanyID diisi bila
// Scope bertipe branch (company induk branch), agar assignment level company berlaku.
type CheckScope struct {
	Scope           valueobject.Scope
	BranchCompanyID string
}
