package valueobject

import (
	"regexp"
	"strings"

	"platform-service/internal/domain/benefit/constant"
	domainerr "platform-service/internal/domain/errors"
)

var benefitCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// BenefitCode — snake_case huruf kecil (mis. "max_employees"), dipakai solution di kode.
type BenefitCode struct {
	value string
}

func NewBenefitCode(raw string) (BenefitCode, error) {
	v := strings.TrimSpace(raw)
	if v == "" || len(v) > constant.CodeMaxLength || !benefitCodePattern.MatchString(v) {
		return BenefitCode{}, domainerr.New(constant.CodeBenefitCodeInvalid)
	}
	return BenefitCode{value: v}, nil
}

func (c BenefitCode) Value() string { return c.value }

type ValueType string

const (
	ValueTypeBoolean ValueType = "boolean"
	ValueTypeInteger ValueType = "integer"
	ValueTypeDecimal ValueType = "decimal"
	ValueTypeString  ValueType = "string"
	ValueTypeJSON    ValueType = "json"
)

func NewValueType(raw string) (ValueType, error) {
	switch v := ValueType(raw); v {
	case ValueTypeBoolean, ValueTypeInteger, ValueTypeDecimal, ValueTypeString, ValueTypeJSON:
		return v, nil
	default:
		return "", domainerr.New(constant.CodeBenefitValueTypeInvalid)
	}
}

func (v ValueType) IsNumeric() bool { return v == ValueTypeInteger || v == ValueTypeDecimal }

type AggregationStrategy string

const (
	AggregationSum      AggregationStrategy = "sum"
	AggregationMax      AggregationStrategy = "max"
	AggregationMin      AggregationStrategy = "min"
	AggregationOverride AggregationStrategy = "override"
	AggregationAnyTrue  AggregationStrategy = "any_true"
)

func NewAggregationStrategy(raw string) (AggregationStrategy, error) {
	switch a := AggregationStrategy(raw); a {
	case AggregationSum, AggregationMax, AggregationMin, AggregationOverride, AggregationAnyTrue:
		return a, nil
	default:
		return "", domainerr.New(constant.CodeBenefitAggregationInvalid)
	}
}

// SupportsValueType — kombinasi strategi agregasi & tipe nilai yang masuk akal.
func (a AggregationStrategy) SupportsValueType(v ValueType) bool {
	switch a {
	case AggregationOverride:
		return true
	case AggregationAnyTrue:
		return v == ValueTypeBoolean
	case AggregationSum, AggregationMax, AggregationMin:
		return v.IsNumeric()
	default:
		return false
	}
}

type Metering string

const (
	MeteringNone    Metering = "none"
	MeteringCounter Metering = "counter"
	MeteringGauge   Metering = "gauge"
)

// NewMetering — kosong → none.
func NewMetering(raw string) (Metering, error) {
	if raw == "" {
		return MeteringNone, nil
	}
	switch m := Metering(raw); m {
	case MeteringNone, MeteringCounter, MeteringGauge:
		return m, nil
	default:
		return "", domainerr.New(constant.CodeBenefitMeteringInvalid)
	}
}

type UsageReset string

const (
	UsageResetNever         UsageReset = "never"
	UsageResetBillingPeriod UsageReset = "billing_period"
	UsageResetMonthly       UsageReset = "monthly"
)

// NewUsageReset — kosong → never.
func NewUsageReset(raw string) (UsageReset, error) {
	if raw == "" {
		return UsageResetNever, nil
	}
	switch r := UsageReset(raw); r {
	case UsageResetNever, UsageResetBillingPeriod, UsageResetMonthly:
		return r, nil
	default:
		return "", domainerr.New(constant.CodeBenefitUsageResetInvalid)
	}
}
