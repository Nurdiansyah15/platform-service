package entity

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"platform-service/internal/domain/benefit/constant"
	"platform-service/internal/domain/benefit/valueobject"
	domainerr "platform-service/internal/domain/errors"
)

// UnlimitedValue — nilai benefit angka yang berarti tanpa batas (keputusan Fase 12).
// Nilai angka lain harus >= 0.
const UnlimitedValue = -1

// Benefit adalah fitur / limit yang bisa diberikan plan atau add-on (spec §2.4).
type Benefit struct {
	ID                  string
	SolutionID          string
	Code                valueobject.BenefitCode
	Name                string
	Description         *string
	ValueType           valueobject.ValueType
	AggregationStrategy valueobject.AggregationStrategy
	Unit                *string
	Metering            valueobject.Metering
	UsageReset          valueobject.UsageReset
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type NewBenefitParams struct {
	SolutionID          string
	Code                string
	Name                string
	Description         *string
	ValueType           string
	AggregationStrategy string
	Unit                *string
	Metering            string // kosong → none
	UsageReset          string // kosong → never
}

func NewBenefit(p NewBenefitParams, now time.Time) (*Benefit, error) {
	code, err := valueobject.NewBenefitCode(p.Code)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(p.Name)
	if name == "" || utf8.RuneCountInString(name) > constant.NameMaxLength {
		return nil, domainerr.New(constant.CodeBenefitNameInvalid)
	}
	unit := trimOptional(p.Unit)
	if unit != nil && utf8.RuneCountInString(*unit) > constant.UnitMaxLength {
		return nil, domainerr.New(constant.CodeBenefitUnitInvalid)
	}
	valueType, err := valueobject.NewValueType(p.ValueType)
	if err != nil {
		return nil, err
	}
	aggregation, err := valueobject.NewAggregationStrategy(p.AggregationStrategy)
	if err != nil {
		return nil, err
	}
	if !aggregation.SupportsValueType(valueType) {
		return nil, domainerr.New(constant.CodeBenefitAggregationInvalid)
	}
	metering, err := valueobject.NewMetering(p.Metering)
	if err != nil {
		return nil, err
	}
	// Pemakaian hanya bisa diukur terhadap limit angka.
	if metering != valueobject.MeteringNone && !valueType.IsNumeric() {
		return nil, domainerr.New(constant.CodeBenefitMeteringInvalid)
	}
	reset, err := valueobject.NewUsageReset(p.UsageReset)
	if err != nil {
		return nil, err
	}
	// usage_reset hanya bermakna untuk counter; gauge tidak pernah di-reset (spec §2.4).
	if metering != valueobject.MeteringCounter && reset != valueobject.UsageResetNever {
		return nil, domainerr.New(constant.CodeBenefitUsageResetInvalid)
	}

	now = now.UTC()
	return &Benefit{
		ID:                  uuid.NewString(),
		SolutionID:          p.SolutionID,
		Code:                code,
		Name:                name,
		Description:         trimOptional(p.Description),
		ValueType:           valueType,
		AggregationStrategy: aggregation,
		Unit:                unit,
		Metering:            metering,
		UsageReset:          reset,
		CreatedAt:           now,
		UpdatedAt:           now,
	}, nil
}

// IsMetered — benefit yang pemakaiannya dilaporkan solution (counter / gauge).
func (b *Benefit) IsMetered() bool {
	return b.Metering != valueobject.MeteringNone
}

// ValidateValue memeriksa nilai JSON benefit (mis. di plan_benefits) sesuai value_type.
func (b *Benefit) ValidateValue(raw json.RawMessage) error {
	invalid := domainerr.New(constant.CodeBenefitValueInvalid)
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return invalid
	}

	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return invalid
	}

	switch b.ValueType {
	case valueobject.ValueTypeBoolean:
		if _, ok := v.(bool); !ok {
			return invalid
		}
	case valueobject.ValueTypeInteger:
		n, ok := v.(json.Number)
		if !ok {
			return invalid
		}
		i, err := n.Int64()
		if err != nil || (i < 0 && i != UnlimitedValue) {
			return invalid
		}
	case valueobject.ValueTypeDecimal:
		n, ok := v.(json.Number)
		if !ok {
			return invalid
		}
		f, err := n.Float64()
		if err != nil || (f < 0 && f != UnlimitedValue) {
			return invalid
		}
	case valueobject.ValueTypeString:
		if _, ok := v.(string); !ok {
			return invalid
		}
	case valueobject.ValueTypeJSON:
		switch v.(type) {
		case map[string]any, []any:
		default:
			return invalid
		}
	default:
		return invalid
	}
	return nil
}

func trimOptional(v *string) *string {
	if v == nil {
		return nil
	}
	t := strings.TrimSpace(*v)
	if t == "" {
		return nil
	}
	return &t
}
