package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	benefitvo "platform-service/internal/domain/benefit/valueobject"
	"platform-service/internal/domain/usage/service"
)

func TestResolvePeriod(t *testing.T) {
	at := time.Date(2026, 2, 14, 23, 30, 0, 0, time.FixedZone("WIB", 7*3600)) // = 14 Feb 16:30 UTC

	never := service.ResolvePeriod(benefitvo.UsageResetNever, at, nil, nil)
	assert.Equal(t, service.NeverResetStart, never.Start)
	assert.Nil(t, never.End)

	monthly := service.ResolvePeriod(benefitvo.UsageResetMonthly, at, nil, nil)
	assert.Equal(t, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), monthly.Start)
	require.NotNil(t, monthly.End)
	assert.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), *monthly.End)

	bs, be := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC), time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	billing := service.ResolvePeriod(benefitvo.UsageResetBillingPeriod, at, &bs, &be)
	assert.Equal(t, bs, billing.Start)
	assert.Equal(t, be, *billing.End)

	fallback := service.ResolvePeriod(benefitvo.UsageResetBillingPeriod, at, nil, nil)
	assert.Equal(t, monthly.Start, fallback.Start, "tanpa periode subscription → bulan kalender")
}
