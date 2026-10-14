package entity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/plan/constant"
	"platform-service/internal/domain/plan/entity"
	pricingconstant "platform-service/internal/domain/pricing/constant"
	"platform-service/internal/testhelper"
)

func TestNewPlan_Defaults(t *testing.T) {
	p, err := entity.NewPlan(entity.NewPlanParams{SolutionID: "s", Code: " Pro ", Name: " Pro "}, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "pro", p.Code.Value())
	assert.Equal(t, "0.00", p.Price.String())
	assert.Equal(t, "IDR", p.Currency.String())
	assert.Equal(t, "monthly", p.BillingCycle.String())
	assert.True(t, p.IsActive)
}

func TestNewPlan_Invalid(t *testing.T) {
	_, err := entity.NewPlan(entity.NewPlanParams{Code: "pro plan", Name: "X"}, time.Now())
	testhelper.AssertDomainError(t, err, constant.CodePlanCodeInvalid)
	_, err = entity.NewPlan(entity.NewPlanParams{Code: "pro", Name: ""}, time.Now())
	testhelper.AssertDomainError(t, err, constant.CodePlanNameInvalid)
	_, err = entity.NewPlan(entity.NewPlanParams{Code: "pro", Name: "X", Price: "-1"}, time.Now())
	testhelper.AssertDomainError(t, err, pricingconstant.CodePriceInvalid)
	_, err = entity.NewPlan(entity.NewPlanParams{Code: "pro", Name: "X", BillingCycle: "weekly"}, time.Now())
	testhelper.AssertDomainError(t, err, pricingconstant.CodeBillingCycleInvalid)
}
