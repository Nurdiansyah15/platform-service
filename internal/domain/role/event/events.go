// Package event mendefinisikan event outbox role (spec §2.9).
package event

import "time"

const (
	TypeRoleCreated            = "role.created"
	TypeRoleUpdated            = "role.updated"
	TypeRoleDeleted            = "role.deleted"
	TypeRolePermissionsChanged = "role.permissions_changed"
)

type RolePayload struct {
	ID         string  `json:"id"`
	SolutionID string  `json:"solution_id"`
	CompanyID  *string `json:"company_id"`
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	IsSystem   bool    `json:"is_system"`
	// Permissions — isi role setelah role.permissions_changed.
	Permissions []PermissionScope `json:"permissions,omitempty"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// PermissionScope — permission role beserta scope_type-nya.
type PermissionScope struct {
	Code      string `json:"code"`
	ScopeType string `json:"scope_type"`
}
