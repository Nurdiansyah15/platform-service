package entity_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	benefitentity "platform-service/internal/domain/benefit/entity"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/usage/constant"
	"platform-service/internal/domain/usage/entity"
	"platform-service/internal/domain/usage/valueobject"
	"platform-service/internal/testhelper"
)

func benefit(t *testing.T, metering, reset string) *benefitentity.Benefit {
	t.Helper()
	b, err := benefitentity.NewBenefit(benefitentity.NewBenefitParams{
		SolutionID: "sol", Code: "x", Name: "X", ValueType: "integer", AggregationStrategy: "sum", Metering: metering, UsageReset: reset,
	}, time.Now())
	require.NoError(t, err)
	return b
}

func qty(t *testing.T, raw string) valueobject.Quantity {
	t.Helper()
	q, err := valueobject.ParseQuantityJSON(json.RawMessage(raw))
	require.NoError(t, err)
	return q
}

func TestQuantity(t *testing.T) {
	assert.Equal(t, "12.5000", qty(t, `12.5`).String())
	assert.Equal(t, "3.1416", qty(t, `"3.1416"`).String())
	assert.True(t, qty(t, `0`).IsZero())
	assert.False(t, qty(t, `0.0001`).IsZero())
	for _, bad := range []string{`-1`, `1.23456`, `"abc"`, `1e3`, ``, `null`} {
		_, err := valueobject.ParseQuantityJSON(json.RawMessage(bad))
		testhelper.AssertDomainError(t, err, constant.CodeUsageQuantityInvalid)
	}
}

func TestNewUsageRecord_Rules(t *testing.T) {
	key, err := valueobject.NewIdempotencyKey("k1")
	require.NoError(t, err)

	tests := []struct {
		name     string
		metering string
		mode     valueobject.Mode
		quantity string
		want     domainerr.Code
	}{
		{"counter delta", "counter", valueobject.ModeDelta, `5`, ""},
		{"counter delta nol", "counter", valueobject.ModeDelta, `0`, constant.CodeUsageQuantityInvalid},
		{"counter absolute", "counter", valueobject.ModeAbsolute, `5`, constant.CodeUsageWrongMode},
		{"gauge absolute", "gauge", valueobject.ModeAbsolute, `0`, ""},
		{"gauge delta", "gauge", valueobject.ModeDelta, `5`, constant.CodeUsageWrongMode},
		{"tidak di-meter", "none", valueobject.ModeDelta, `5`, constant.CodeUsageNotMetered},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entity.NewUsageRecord(entity.NewUsageRecordParams{
				CompanyID: "t", Benefit: benefit(t, tt.metering, ""), Mode: tt.mode, Quantity: qty(t, tt.quantity), IdempotencyKey: key,
			}, time.Now())
			if tt.want == "" {
				assert.NoError(t, err)
				return
			}
			testhelper.AssertDomainError(t, err, tt.want)
		})
	}

	_, err = valueobject.NewIdempotencyKey(" ")
	testhelper.AssertDomainError(t, err, constant.CodeUsageIdempotencyKeyInvalid)
}
