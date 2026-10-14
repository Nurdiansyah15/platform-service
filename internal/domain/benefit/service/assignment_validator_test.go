package service_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/benefit/constant"
	"platform-service/internal/domain/benefit/entity"
	"platform-service/internal/domain/benefit/service"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/testhelper"
)

type fakeBenefits map[string]*entity.Benefit

func (f fakeBenefits) Save(context.Context, *entity.Benefit) error { return nil }
func (f fakeBenefits) FindByID(_ context.Context, id string) (*entity.Benefit, error) {
	if b, ok := f[id]; ok {
		return b, nil
	}
	return nil, domainerr.New(constant.CodeBenefitNotFound)
}
func (f fakeBenefits) FindByCode(context.Context, string, string) (*entity.Benefit, error) {
	return nil, domainerr.New(constant.CodeBenefitNotFound)
}
func (f fakeBenefits) ListBySolution(context.Context, string) ([]*entity.Benefit, error) {
	return nil, nil
}
func (f fakeBenefits) FindByIDs(_ context.Context, ids []string) ([]*entity.Benefit, error) {
	var out []*entity.Benefit
	for _, id := range ids {
		if b, ok := f[id]; ok {
			out = append(out, b)
		}
	}
	return out, nil
}

func benefit(t *testing.T, solutionID, code, valueType, aggregation string) *entity.Benefit {
	t.Helper()
	b, err := entity.NewBenefit(entity.NewBenefitParams{
		SolutionID: solutionID, Code: code, Name: code, ValueType: valueType, AggregationStrategy: aggregation,
	}, time.Now())
	require.NoError(t, err)
	return b
}

func TestAssignmentValidator(t *testing.T) {
	maxEmp := benefit(t, "sol-a", "max_employees", "integer", "sum")
	payroll := benefit(t, "sol-a", "payroll_module", "boolean", "any_true")
	foreign := benefit(t, "sol-b", "other", "integer", "sum")
	v := service.NewAssignmentValidator(fakeBenefits{maxEmp.ID: maxEmp, payroll.ID: payroll, foreign.ID: foreign})
	ctx := context.Background()

	got, err := v.Validate(ctx, "sol-a", []service.Assignment{
		{BenefitID: maxEmp.ID, Value: json.RawMessage(`200`)},
		{BenefitID: payroll.ID, Value: json.RawMessage(`true`)},
	})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "max_employees", got[0].Benefit.Code.Value())

	empty, err := v.Validate(ctx, "sol-a", nil)
	require.NoError(t, err)
	assert.Empty(t, empty)

	tests := []struct {
		name  string
		items []service.Assignment
		want  domainerr.Code
	}{
		{"duplikat", []service.Assignment{{BenefitID: maxEmp.ID, Value: json.RawMessage(`1`)}, {BenefitID: maxEmp.ID, Value: json.RawMessage(`2`)}}, constant.CodeBenefitAssignmentDuplicate},
		{"tak dikenal", []service.Assignment{{BenefitID: "nope", Value: json.RawMessage(`1`)}}, constant.CodeBenefitAssignmentUnknown},
		{"solution lain", []service.Assignment{{BenefitID: foreign.ID, Value: json.RawMessage(`1`)}}, constant.CodeBenefitSolutionMismatch},
		{"nilai salah tipe", []service.Assignment{{BenefitID: payroll.ID, Value: json.RawMessage(`"yes"`)}}, constant.CodeBenefitValueInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := v.Validate(ctx, "sol-a", tt.items)
			testhelper.AssertDomainError(t, err, tt.want)
		})
	}
}
