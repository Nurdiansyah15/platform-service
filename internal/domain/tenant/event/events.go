// Package event mendefinisikan event outbox domain tenant (spec §2.9).
package event

import "time"

const (
	TypeTenantMemberAdded = "tenant_member.added"
)

// TenantMemberPayload adalah payload tenant_member.* (dikirim ke webhook).
type TenantMemberPayload struct {
	TenantID string    `json:"tenant_id"`
	UserID   string    `json:"user_id"`
	Status   string    `json:"status"`
	JoinedAt time.Time `json:"joined_at"`
}
