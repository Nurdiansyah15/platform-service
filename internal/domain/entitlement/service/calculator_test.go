package service_test

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	benefitentity "platform-service/internal/domain/benefit/entity"
	"platform-service/internal/domain/entitlement/entity"
	"platform-service/internal/domain/entitlement/service"
)

func benefit(t *testing.T, valueType, aggregation string) *benefitentity.Benefit {
	t.Helper()
	b, err := benefitentity.NewBenefit(benefitentity.NewBenefitParams{
		SolutionID: "s", Code: "b", Name: "B", ValueType: valueType, AggregationStrategy: aggregation,
	}, time.Now())
	require.NoError(t, err)
	return b
}

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func plan(v string) entity.SnapshotValue {
	return entity.SnapshotValue{Value: json.RawMessage(v), Quantity: 1}
}
func addon(v string, qty int, minutes int) entity.SnapshotValue {
	return entity.SnapshotValue{Value: json.RawMessage(v), FromAddon: true, Quantity: qty, AttachedAt: t0.Add(time.Duration(minutes) * time.Minute)}
}

func TestCalculate(t *testing.T) {
	tests := []struct {
		name        string
		valueType   string
		aggregation string
		values      []entity.SnapshotValue
		wantValue   string
		unlimited   bool
	}{
		{"sum plan saja", "integer", "sum", []entity.SnapshotValue{plan(`200`)}, `200`, false},
		{"sum plan + add-on × qty", "integer", "sum", []entity.SnapshotValue{plan(`200`), addon(`50`, 2, 1), addon(`10`, 1, 2)}, `310`, false},
		{"sum decimal", "decimal", "sum", []entity.SnapshotValue{plan(`1.5`), addon(`0.25`, 2, 1)}, `2`, false},
		{"sum decimal pecahan", "decimal", "sum", []entity.SnapshotValue{plan(`1.25`), addon(`0.5`, 1, 1)}, `1.75`, false},
		{"sum dengan unlimited", "integer", "sum", []entity.SnapshotValue{plan(`200`), addon(`-1`, 1, 1)}, `-1`, true},
		{"sum add-on tanpa plan", "integer", "sum", []entity.SnapshotValue{addon(`5`, 3, 1)}, `15`, false},
		{"max", "integer", "max", []entity.SnapshotValue{plan(`10`), addon(`25`, 5, 1), addon(`20`, 1, 2)}, `25`, false},
		{"max unlimited", "integer", "max", []entity.SnapshotValue{plan(`-1`), addon(`25`, 1, 1)}, `-1`, true},
		{"min abaikan unlimited", "integer", "min", []entity.SnapshotValue{plan(`-1`), addon(`25`, 1, 1), addon(`30`, 1, 2)}, `25`, false},
		{"min semua unlimited", "integer", "min", []entity.SnapshotValue{plan(`-1`)}, `-1`, true},
		{"override plan saja", "integer", "override", []entity.SnapshotValue{plan(`7`)}, `7`, false},
		{"override add-on terbaru", "integer", "override", []entity.SnapshotValue{addon(`9`, 1, 5), plan(`7`), addon(`8`, 1, 2)}, `9`, false},
		{"any_true", "boolean", "any_true", []entity.SnapshotValue{plan(`false`), addon(`true`, 1, 1)}, `true`, false},
		{"any_true semua false", "boolean", "any_true", []entity.SnapshotValue{plan(`false`)}, `false`, false},
		{"boolean override", "boolean", "override", []entity.SnapshotValue{plan(`true`), addon(`false`, 1, 1)}, `false`, false},
		{"string override", "string", "override", []entity.SnapshotValue{plan(`"silver"`), addon(`"gold"`, 1, 1)}, `"gold"`, false},
		{"json override plan", "json", "override", []entity.SnapshotValue{plan(`{"a":1}`)}, `{"a":1}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.Calculate(benefit(t, tt.valueType, tt.aggregation), tt.values)
			assert.True(t, got.Granted)
			assert.JSONEq(t, tt.wantValue, string(got.Value))
			assert.Equal(t, tt.unlimited, got.Unlimited)
			numeric := tt.valueType == "integer" || tt.valueType == "decimal"
			if tt.unlimited || !numeric {
				assert.Nil(t, got.Limit)
			} else {
				require.NotNil(t, got.Limit)
				assert.Equal(t, tt.wantValue, service.FormatRat(got.Limit))
			}
		})
	}
}

func TestCalculate_NotGranted(t *testing.T) {
	got := service.Calculate(benefit(t, "integer", "sum"), nil)
	assert.False(t, got.Granted)
	assert.JSONEq(t, `null`, string(got.Value))
	assert.Nil(t, got.Limit)
}

func TestRemaining(t *testing.T) {
	rem, reached := service.Remaining(big.NewRat(10, 1), big.NewRat(3, 1))
	assert.Equal(t, "7", service.FormatRat(rem))
	assert.False(t, reached)

	rem, reached = service.Remaining(big.NewRat(10, 1), big.NewRat(10, 1))
	assert.Equal(t, "0", service.FormatRat(rem))
	assert.True(t, reached)

	rem, reached = service.Remaining(big.NewRat(10, 1), big.NewRat(12, 1))
	assert.Equal(t, "0", service.FormatRat(rem), "tidak negatif")
	assert.True(t, reached)
}
