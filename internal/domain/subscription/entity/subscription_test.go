package entity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pricingvo "platform-service/internal/domain/pricing/valueobject"
	"platform-service/internal/domain/subscription/constant"
	"platform-service/internal/domain/subscription/entity"
	"platform-service/internal/domain/subscription/valueobject"
	"platform-service/internal/testhelper"
)

func terms(t *testing.T, planID, cycle, price string) entity.PlanTerms {
	t.Helper()
	p, err := pricingvo.NewPrice(price)
	require.NoError(t, err)
	c, err := pricingvo.NewCurrency("")
	require.NoError(t, err)
	bc, err := pricingvo.NewBillingCycle(cycle)
	require.NoError(t, err)
	return entity.PlanTerms{PlanID: planID, SolutionID: "sol", BillingCycle: bc, Price: p, Currency: c}
}

var jan31 = time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC)

func TestNewSubscription_Period(t *testing.T) {
	s, err := entity.NewSubscription("t1", terms(t, "basic", "monthly", "100000"), "", jan31, jan31)
	require.NoError(t, err)
	assert.Equal(t, valueobject.StatusActive, s.Status)
	assert.Equal(t, "100000.00", s.PriceAtPurchase.String())
	assert.Equal(t, time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC), *s.CurrentPeriodEnd, "31 Jan + 1 bulan dijepit ke 28 Feb")

	y, err := entity.NewSubscription("t1", terms(t, "basic", "yearly", "1"), "trialing", jan31, jan31)
	require.NoError(t, err)
	assert.Equal(t, valueobject.StatusTrialing, y.Status)
	assert.Equal(t, time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC), *y.CurrentPeriodEnd)

	leap := time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC)
	ly, err := entity.NewSubscription("t1", terms(t, "basic", "yearly", "1"), "", leap, leap)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2029, 2, 28, 0, 0, 0, 0, time.UTC), *ly.CurrentPeriodEnd, "29 Feb + 1 tahun dijepit ke 28 Feb")

	mid := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	m, err := entity.NewSubscription("t1", terms(t, "basic", "monthly", "1"), "", mid, mid)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC), *m.CurrentPeriodEnd)

	_, err = entity.NewSubscription("t1", terms(t, "basic", "monthly", "1"), "canceled", jan31, jan31)
	testhelper.AssertDomainError(t, err, constant.CodeSubscriptionStatusInvalid)
}

func TestSubscription_ChangePlanAndCancel(t *testing.T) {
	s, err := entity.NewSubscription("t1", terms(t, "basic", "monthly", "100000"), "", jan31, jan31)
	require.NoError(t, err)

	testhelper.AssertDomainError(t, s.ChangePlan(terms(t, "basic", "monthly", "1"), jan31), constant.CodeSubscriptionSamePlan)

	later := jan31.Add(48 * time.Hour)
	require.NoError(t, s.ChangePlan(terms(t, "pro", "yearly", "1000000"), later))
	assert.Equal(t, "pro", s.PlanID)
	assert.Equal(t, "1000000.00", s.PriceAtPurchase.String())
	assert.Equal(t, later, *s.CurrentPeriodStart)

	require.NoError(t, s.Cancel(later))
	assert.Equal(t, valueobject.StatusCanceled, s.Status)
	testhelper.AssertDomainError(t, s.Cancel(later), constant.CodeSubscriptionNotLive)
	testhelper.AssertDomainError(t, s.ChangePlan(terms(t, "basic", "monthly", "1"), later), constant.CodeSubscriptionNotLive)
}

func TestSubscription_AttachAddon(t *testing.T) {
	s, err := entity.NewSubscription("t1", terms(t, "basic", "monthly", "1"), "", jan31, jan31)
	require.NoError(t, err)
	price, _ := pricingvo.NewPrice("25000")
	cur, _ := pricingvo.NewCurrency("")
	addon := entity.AddonTerms{AddonID: "payroll", SolutionID: "sol", Price: price, Currency: cur}

	_, err = s.AttachAddon(addon, 0, jan31)
	testhelper.AssertDomainError(t, err, constant.CodeSubscriptionAddonQuantity)
	_, err = s.AttachAddon(entity.AddonTerms{AddonID: "x", SolutionID: "other"}, 1, jan31)
	testhelper.AssertDomainError(t, err, constant.CodeSubscriptionAddonUnavailable)

	a, err := s.AttachAddon(addon, 2, jan31)
	require.NoError(t, err)
	assert.Equal(t, 2, a.Quantity)
	assert.Equal(t, "25000.00", a.UnitPriceAtPurchase.String())

	testhelper.AssertDomainError(t, a.Reactivate(addon, 1, jan31), constant.CodeSubscriptionAddonAlreadyActive)
	a.Status = entity.AddonStatusCanceled
	require.NoError(t, a.Reactivate(addon, 3, jan31))
	assert.Equal(t, entity.AddonStatusActive, a.Status)
	assert.Equal(t, 3, a.Quantity)

	require.NoError(t, s.Cancel(jan31))
	_, err = s.AttachAddon(addon, 1, jan31)
	testhelper.AssertDomainError(t, err, constant.CodeSubscriptionNotLive)
}
