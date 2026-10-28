// Package event mendefinisikan event outbox organisasi (spec §2.9).
package event

import "time"

const (
	TypeCompanyCreated     = "company.created"
	TypeBranchCreated      = "branch.created"
	TypeCompanyMemberAdded = "company_member.added"
)

type CompanyPayload struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

type BranchPayload struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"company_id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CompanyMemberPayload adalah payload company_member.* (dikirim ke webhook).
type CompanyMemberPayload struct {
	CompanyID string    `json:"company_id"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	JoinedAt  time.Time `json:"joined_at"`
}
