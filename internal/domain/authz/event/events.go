// Package event mendefinisikan event outbox otorisasi (spec §2.9).
package event

import "time"

const (
	TypeRoleAssignmentCreated = "role_assignment.created"
	TypeRoleAssignmentDeleted = "role_assignment.deleted"
)

type RoleAssignmentPayload struct {
	ID         string `json:"id"`
	CompanyID  string `json:"company_id"`
	UserID     string `json:"user_id"`
	RoleID     string `json:"role_id"`
	RoleCode   string `json:"role_code"`
	SolutionID string `json:"solution_id"`
	// ScopeValues — nilai scope assignment per scope_type ({} bila role hanya ber-scope company).
	ScopeValues map[string]string `json:"scope_values"`
	ScopeLabels map[string]string `json:"scope_labels,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}
