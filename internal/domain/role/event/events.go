// Package event mendefinisikan event outbox role (spec §2.9).
package event

import "time"

const (
	TypeRoleCreated            = "role.created"
	TypeRolePermissionsChanged = "role.permissions_changed"
)

type RolePayload struct {
	ID              string    `json:"id"`
	SolutionID      string    `json:"solution_id"`
	CompanyID       *string   `json:"company_id"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	IsSystem        bool      `json:"is_system"`
	PermissionCodes []string  `json:"permission_codes,omitempty"`
	UpdatedAt       time.Time `json:"updated_at"`
}
