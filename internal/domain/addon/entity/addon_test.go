package entity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/addon/constant"
	"platform-service/internal/domain/addon/entity"
	pricingconstant "platform-service/internal/domain/pricing/constant"
	"platform-service/internal/testhelper"
)

func TestNewAddon(t *testing.T) {
	a, err := entity.NewAddon(entity.NewAddonParams{SolutionID: "s", Code: " Payroll ", Name: "Payroll", Price: "25000"}, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "payroll", a.Code.Value())
	assert.Equal(t, "25000.00", a.Price.String())
	assert.True(t, a.IsActive)

	_, err = entity.NewAddon(entity.NewAddonParams{Code: "pay roll", Name: "X"}, time.Now())
	testhelper.AssertDomainError(t, err, constant.CodeAddonCodeInvalid)
	_, err = entity.NewAddon(entity.NewAddonParams{Code: "payroll", Name: " "}, time.Now())
	testhelper.AssertDomainError(t, err, constant.CodeAddonNameInvalid)
	_, err = entity.NewAddon(entity.NewAddonParams{Code: "payroll", Name: "X", Currency: "1DR"}, time.Now())
	testhelper.AssertDomainError(t, err, pricingconstant.CodeCurrencyInvalid)
}
