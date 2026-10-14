// Package event mendefinisikan event outbox organisasi (spec §2.9).
package event

import "time"

const (
	TypeCompanyCreated = "company.created"
	TypeBranchCreated  = "branch.created"
)

type CompanyPayload struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

type BranchPayload struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	CompanyID string    `json:"company_id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}
