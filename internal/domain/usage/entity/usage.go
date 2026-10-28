package entity

import (
	"time"

	"github.com/google/uuid"

	benefitentity "platform-service/internal/domain/benefit/entity"
	benefitvo "platform-service/internal/domain/benefit/valueobject"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/usage/constant"
	"platform-service/internal/domain/usage/valueobject"
)

// UsageRecord adalah satu laporan pemakaian (append-only).
type UsageRecord struct {
	ID                 string
	CompanyID          string
	SolutionID         string
	BenefitID          string
	Mode               valueobject.Mode
	Quantity           valueobject.Quantity
	IdempotencyKey     valueobject.IdempotencyKey
	ReportedByClientID string
	OccurredAt         time.Time
	CreatedAt          time.Time
}

type NewUsageRecordParams struct {
	CompanyID          string
	Benefit            *benefitentity.Benefit
	Mode               valueobject.Mode
	Quantity           valueobject.Quantity
	IdempotencyKey     valueobject.IdempotencyKey
	ReportedByClientID string
	OccurredAt         time.Time
}

// NewUsageRecord menerapkan aturan spec §2.10:
//   - benefit metering none menolak laporan
//   - counter ← delta (harus > 0), gauge ← absolute (≥ 0)
func NewUsageRecord(p NewUsageRecordParams, now time.Time) (*UsageRecord, error) {
	switch p.Benefit.Metering {
	case benefitvo.MeteringNone:
		return nil, domainerr.New(constant.CodeUsageNotMetered)
	case benefitvo.MeteringCounter:
		if p.Mode != valueobject.ModeDelta {
			return nil, domainerr.New(constant.CodeUsageWrongMode)
		}
		if p.Quantity.IsZero() {
			return nil, domainerr.New(constant.CodeUsageQuantityInvalid)
		}
	case benefitvo.MeteringGauge:
		if p.Mode != valueobject.ModeAbsolute {
			return nil, domainerr.New(constant.CodeUsageWrongMode)
		}
	}
	occurred := p.OccurredAt
	if occurred.IsZero() {
		occurred = now
	}
	return &UsageRecord{
		ID:                 uuid.NewString(),
		CompanyID:          p.CompanyID,
		SolutionID:         p.Benefit.SolutionID,
		BenefitID:          p.Benefit.ID,
		Mode:               p.Mode,
		Quantity:           p.Quantity,
		IdempotencyKey:     p.IdempotencyKey,
		ReportedByClientID: p.ReportedByClientID,
		OccurredAt:         occurred.UTC(),
		CreatedAt:          now.UTC(),
	}, nil
}

// SamePayload — laporan ulang dengan idempotency key yang sama dianggap replay
// hanya bila isinya identik.
func (r *UsageRecord) SamePayload(o *UsageRecord) bool {
	return r.CompanyID == o.CompanyID && r.BenefitID == o.BenefitID && r.Mode == o.Mode && r.Quantity == o.Quantity
}

// Period adalah rentang pemakaian [Start, End); End nil = tidak pernah di-reset.
type Period struct {
	Start time.Time
	End   *time.Time
}

// CompanyUsage adalah pemakaian terkini satu benefit dalam satu periode.
type CompanyUsage struct {
	CompanyID      string
	SolutionID     string
	BenefitID      string
	Period         Period
	Used           string
	LastOccurredAt *time.Time
	UpdatedAt      time.Time
}
