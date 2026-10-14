// Package event mendefinisikan event outbox entitlement / usage (spec §2.9).
package event

import "time"

const TypeUsageLimitReached = "usage.limit_reached"

// UsageLimitReachedPayload dikirim saat pemakaian benefit melewati limit (spec §2.10).
type UsageLimitReachedPayload struct {
	TenantID    string     `json:"tenant_id"`
	SolutionID  string     `json:"solution_id"`
	BenefitCode string     `json:"benefit_code"`
	Limit       string     `json:"limit"`
	Used        string     `json:"used"`
	PeriodStart time.Time  `json:"period_start"`
	PeriodEnd   *time.Time `json:"period_end"`
	OccurredAt  time.Time  `json:"occurred_at"`
}
