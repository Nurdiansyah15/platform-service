package dto

import (
	"encoding/json"
	"time"
)

type EntitlementBenefitParam struct {
	CompanyID   string `uri:"companyId" binding:"required,uuid"`
	SolutionID  string `uri:"solutionId" binding:"required,uuid"`
	BenefitCode string `uri:"benefitCode" binding:"required,max=100"`
}

// EntitlementsResponse — hak efektif company atas seluruh benefit solution.
type EntitlementsResponse struct {
	CompanyID  string `json:"company_id"`
	SolutionID string `json:"solution_id"`
	// Subscription — null bila company tidak berlangganan (semua benefit granted=false).
	Subscription *EntitlementSubscription `json:"subscription"`
	Entitlements []EntitlementResponse    `json:"entitlements"`
}

type EntitlementSubscription struct {
	ID               string     `json:"id"`
	Status           string     `json:"status"`
	PlanID           string     `json:"plan_id"`
	CurrentPeriodEnd *time.Time `json:"current_period_end"`
}

type EntitlementResponse struct {
	BenefitCode         string  `json:"benefit_code"`
	Name                string  `json:"name"`
	ValueType           string  `json:"value_type"`
	AggregationStrategy string  `json:"aggregation_strategy"`
	Unit                *string `json:"unit"`
	Metering            string  `json:"metering"`
	// Granted — benefit diberikan plan / add-on aktif.
	Granted bool `json:"granted"`
	// Value — nilai efektif; -1 = unlimited; null bila tidak granted.
	Value     json.RawMessage `json:"value" swaggertype:"object"`
	Unlimited bool            `json:"unlimited"`
	// Limit / Used / Remaining hanya untuk benefit angka yang di-meter (counter / gauge).
	Limit        *string    `json:"limit"`
	Used         *string    `json:"used"`
	Remaining    *string    `json:"remaining"`
	LimitReached bool       `json:"limit_reached"`
	PeriodStart  *time.Time `json:"period_start"`
	PeriodEnd    *time.Time `json:"period_end"`
}
