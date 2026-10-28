// Package event mendefinisikan event outbox otorisasi (spec §2.9).
package event

import "time"

const TypeRoleAssignmentCreated = "role_assignment.created"

type RoleAssignmentPayload struct {
	ID         string    `json:"id"`
	CompanyID  string    `json:"company_id"`
	UserID     string    `json:"user_id"`
	RoleID     string    `json:"role_id"`
	RoleCode   string    `json:"role_code"`
	SolutionID string    `json:"solution_id"`
	ScopeType  string    `json:"scope_type"`
	ScopeID    string    `json:"scope_id"`
	CreatedAt  time.Time `json:"created_at"`
}
