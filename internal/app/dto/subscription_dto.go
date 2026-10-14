package dto

import "time"

type CreateSubscriptionRequest struct {
	PlanID string `json:"plan_id" binding:"required,uuid"`
	// Status awal: active (default) atau trialing.
	Status string `json:"status" binding:"omitempty,oneof=active trialing"`
	// CurrentPeriodStart — default sekarang. current_period_end dihitung dari billing_cycle plan.
	CurrentPeriodStart *time.Time `json:"current_period_start"`
}

type ListSubscriptionQuery struct {
	Status *string `form:"status" binding:"omitempty,oneof=trialing active past_due canceled expired"`
	PaginationParams
}

type SubscriptionIDParam struct {
	SubscriptionID string `uri:"subscriptionId" binding:"required,uuid"`
}

type ChangePlanRequest struct {
	PlanID string `json:"plan_id" binding:"required,uuid"`
}

type AddSubscriptionAddonRequest struct {
	AddonID string `json:"addon_id" binding:"required,uuid"`
	// Quantity — default 1.
	Quantity *int `json:"quantity" binding:"omitempty,min=1,max=1000000"`
}

type SubscriptionResponse struct {
	ID                 string     `json:"id"`
	TenantID           string     `json:"tenant_id"`
	SolutionID         string     `json:"solution_id"`
	SolutionCode       string     `json:"solution_code"`
	PlanID             string     `json:"plan_id"`
	PlanCode           string     `json:"plan_code"`
	PlanName           string     `json:"plan_name"`
	Status             string     `json:"status"`
	BillingCycle       string     `json:"billing_cycle"`
	CurrentPeriodStart *time.Time `json:"current_period_start"`
	CurrentPeriodEnd   *time.Time `json:"current_period_end"`
	PriceAtPurchase    string     `json:"price_at_purchase" example:"150000.00"`
	Currency           string     `json:"currency"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type SubscriptionAddonResponse struct {
	ID                  string    `json:"id"`
	SubscriptionID      string    `json:"subscription_id"`
	AddonID             string    `json:"addon_id"`
	AddonCode           string    `json:"addon_code"`
	AddonName           string    `json:"addon_name"`
	Quantity            int       `json:"quantity"`
	UnitPriceAtPurchase string    `json:"unit_price_at_purchase" example:"25000.00"`
	Currency            string    `json:"currency"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
