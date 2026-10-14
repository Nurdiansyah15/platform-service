package entity_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/benefit/constant"
	"platform-service/internal/domain/benefit/entity"
	"platform-service/internal/domain/benefit/valueobject"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/testhelper"
)

func params(valueType, aggregation, metering, reset string) entity.NewBenefitParams {
	return entity.NewBenefitParams{
		SolutionID: "s", Code: "max_employees", Name: "Max Employees",
		ValueType: valueType, AggregationStrategy: aggregation, Metering: metering, UsageReset: reset,
	}
}

func TestNewBenefit_Defaults(t *testing.T) {
	unit := "  orang "
	p := params("integer", "sum", "", "")
	p.Unit = &unit
	b, err := entity.NewBenefit(p, time.Now())
	require.NoError(t, err)
	assert.Equal(t, valueobject.MeteringNone, b.Metering)
	assert.Equal(t, valueobject.UsageResetNever, b.UsageReset)
	assert.Equal(t, "orang", *b.Unit)
	assert.False(t, b.IsMetered())
}

func TestNewBenefit_Combinations(t *testing.T) {
	tests := []struct {
		name string
		p    entity.NewBenefitParams
		want domainerr.Code // kosong = valid
	}{
		{"boolean any_true", params("boolean", "any_true", "", ""), ""},
		{"boolean override", params("boolean", "override", "", ""), ""},
		{"boolean sum", params("boolean", "sum", "", ""), constant.CodeBenefitAggregationInvalid},
		{"integer max", params("integer", "max", "", ""), ""},
		{"integer any_true", params("integer", "any_true", "", ""), constant.CodeBenefitAggregationInvalid},
		{"decimal min", params("decimal", "min", "", ""), ""},
		{"string override", params("string", "override", "", ""), ""},
		{"string sum", params("string", "sum", "", ""), constant.CodeBenefitAggregationInvalid},
		{"json override", params("json", "override", "", ""), ""},
		{"counter monthly", params("integer", "sum", "counter", "monthly"), ""},
		{"counter billing_period", params("decimal", "sum", "counter", "billing_period"), ""},
		{"gauge never", params("integer", "max", "gauge", "never"), ""},
		{"gauge monthly", params("integer", "max", "gauge", "monthly"), constant.CodeBenefitUsageResetInvalid},
		{"none monthly", params("integer", "sum", "none", "monthly"), constant.CodeBenefitUsageResetInvalid},
		{"boolean counter", params("boolean", "any_true", "counter", ""), constant.CodeBenefitMeteringInvalid},
		{"value_type tak dikenal", params("float", "sum", "", ""), constant.CodeBenefitValueTypeInvalid},
		{"metering tak dikenal", params("integer", "sum", "hourly", ""), constant.CodeBenefitMeteringInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entity.NewBenefit(tt.p, time.Now())
			if tt.want == "" {
				assert.NoError(t, err)
				return
			}
			testhelper.AssertDomainError(t, err, tt.want)
		})
	}
}

func TestNewBenefit_InvalidCode(t *testing.T) {
	for _, code := range []string{"", "MaxEmployees", "max-employees", "1st_limit", "max employees"} {
		p := params("integer", "sum", "", "")
		p.Code = code
		_, err := entity.NewBenefit(p, time.Now())
		testhelper.AssertDomainError(t, err, constant.CodeBenefitCodeInvalid)
	}
}

func TestBenefit_ValidateValue(t *testing.T) {
	tests := []struct {
		valueType string
		value     string
		valid     bool
	}{
		{"boolean", `true`, true},
		{"boolean", `"true"`, false},
		{"boolean", `1`, false},
		{"integer", `200`, true},
		{"integer", `-1`, true},
		{"integer", `-5`, false},
		{"decimal", `-1`, true},
		{"decimal", `-0.5`, false},
		{"integer", `2.5`, false},
		{"integer", `"200"`, false},
		{"integer", `99999999999999999999`, false},
		{"decimal", `2.5`, true},
		{"decimal", `3`, true},
		{"decimal", `null`, false},
		{"string", `"gold"`, true},
		{"string", `5`, false},
		{"json", `{"a":1}`, true},
		{"json", `[1,2]`, true},
		{"json", `"x"`, false},
		{"json", `{bukan json`, false},
		{"integer", ``, false},
	}
	for _, tt := range tests {
		t.Run(tt.valueType+" "+tt.value, func(t *testing.T) {
			aggregation := "override"
			b, err := entity.NewBenefit(params(tt.valueType, aggregation, "", ""), time.Now())
			require.NoError(t, err)
			err = b.ValidateValue(json.RawMessage(tt.value))
			if tt.valid {
				assert.NoError(t, err)
			} else {
				testhelper.AssertDomainError(t, err, constant.CodeBenefitValueInvalid)
			}
		})
	}
}
