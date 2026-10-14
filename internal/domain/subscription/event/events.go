// Package event mendefinisikan event outbox subscription (spec §2.9).
package event

import "time"

const (
	TypeSubscriptionCreated       = "subscription.created"
	TypeSubscriptionPlanChanged   = "subscription.plan_changed"
	TypeSubscriptionStatusChanged = "subscription.status_changed"
	TypeSubscriptionAddonAdded    = "subscription_addon.added"
	TypeEntitlementChanged        = "entitlement.changed"
)

type SubscriptionPayload struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	SolutionID     string    `json:"solution_id"`
	PlanID         string    `json:"plan_id"`
	Status         string    `json:"status"`
	PreviousPlanID string    `json:"previous_plan_id,omitempty"`
	PreviousStatus string    `json:"previous_status,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SubscriptionAddonPayload struct {
	ID             string    `json:"id"`
	SubscriptionID string    `json:"subscription_id"`
	TenantID       string    `json:"tenant_id"`
	SolutionID     string    `json:"solution_id"`
	AddonID        string    `json:"addon_id"`
	Quantity       int       `json:"quantity"`
	Status         string    `json:"status"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// EntitlementChangedPayload memberi tahu solution agar memuat ulang entitlement tenant.
type EntitlementChangedPayload struct {
	TenantID       string    `json:"tenant_id"`
	SolutionID     string    `json:"solution_id"`
	SubscriptionID string    `json:"subscription_id"`
	Reason         string    `json:"reason"`
	OccurredAt     time.Time `json:"occurred_at"`
}
