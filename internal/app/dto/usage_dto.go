package dto

import (
	"encoding/json"
	"time"
)

type TenantSolutionParam struct {
	TenantID   string `uri:"tenantId" binding:"required,uuid"`
	SolutionID string `uri:"solutionId" binding:"required,uuid"`
}

type TenantSolutionBenefitParam struct {
	TenantID    string `uri:"tenantId" binding:"required,uuid"`
	SolutionID  string `uri:"solutionId" binding:"required,uuid"`
	BenefitCode string `uri:"benefitCode" binding:"required,max=100"`
}

// ReportUsageDeltaRequest — laporan counter (POST .../usage).
type ReportUsageDeltaRequest struct {
	BenefitCode string `json:"benefit_code" binding:"required,max=100"`
	// Quantity — angka atau string angka > 0, maks 4 desimal.
	Quantity       json.RawMessage `json:"quantity" binding:"required" swaggertype:"string" example:"1"`
	IdempotencyKey string          `json:"idempotency_key" binding:"required,max=150"`
	// OccurredAt — default waktu server.
	OccurredAt *time.Time `json:"occurred_at"`
}

// ReportUsageAbsoluteRequest — laporan gauge (PUT .../usage/{benefitCode}).
type ReportUsageAbsoluteRequest struct {
	// Quantity — nilai absolut terkini (angka atau string angka ≥ 0, maks 4 desimal).
	Quantity       json.RawMessage `json:"quantity" binding:"required" swaggertype:"string" example:"120"`
	IdempotencyKey string          `json:"idempotency_key" binding:"required,max=150"`
	OccurredAt     *time.Time      `json:"occurred_at"`
}

type UsageReportResponse struct {
	BenefitCode string     `json:"benefit_code"`
	Mode        string     `json:"mode"`
	PeriodStart time.Time  `json:"period_start"`
	PeriodEnd   *time.Time `json:"period_end"`
	// Used — pemakaian terkini pada periode tersebut (string presisi 4 desimal).
	Used string `json:"used" example:"15.0000"`
	// Replayed — true bila idempotency_key sudah pernah diterima (tidak diterapkan ulang).
	Replayed   bool      `json:"replayed"`
	OccurredAt time.Time `json:"occurred_at"`
}
