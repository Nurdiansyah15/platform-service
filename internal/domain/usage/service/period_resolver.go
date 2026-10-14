// Package service berisi domain service usage.
package service

import (
	"time"

	benefitvo "platform-service/internal/domain/benefit/valueobject"
	"platform-service/internal/domain/usage/entity"
)

// NeverResetStart adalah period_start tetap untuk benefit usage_reset = never (spec §2.10).
var NeverResetStart = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

// ResolvePeriod menentukan periode pemakaian untuk laporan pada occurredAt:
//   - never: periode tetap tanpa akhir
//   - monthly: bulan kalender (UTC) dari occurredAt
//   - billing_period: periode subscription saat ini (billingStart / billingEnd);
//     bila tidak tersedia, jatuh ke bulan kalender
func ResolvePeriod(reset benefitvo.UsageReset, occurredAt time.Time, billingStart, billingEnd *time.Time) entity.Period {
	switch reset {
	case benefitvo.UsageResetMonthly:
		return calendarMonth(occurredAt)
	case benefitvo.UsageResetBillingPeriod:
		if billingStart != nil && billingEnd != nil {
			start, end := billingStart.UTC(), billingEnd.UTC()
			return entity.Period{Start: start, End: &end}
		}
		return calendarMonth(occurredAt)
	default:
		return entity.Period{Start: NeverResetStart}
	}
}

func calendarMonth(t time.Time) entity.Period {
	t = t.UTC()
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	return entity.Period{Start: start, End: &end}
}
