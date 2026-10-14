package dto

import (
	"encoding/json"
	"time"
)

type CreatePlanRequest struct {
	Code        string  `json:"code" binding:"required,max=100"`
	Name        string  `json:"name" binding:"required,max=255"`
	Description *string `json:"description"`
	// Price — angka JSON atau string angka, maks 2 desimal (mis. 150000 atau "150000.50").
	Price        json.RawMessage `json:"price" swaggertype:"string" example:"150000.00"`
	Currency     string          `json:"currency" binding:"omitempty,len=3"`
	BillingCycle string          `json:"billing_cycle" binding:"omitempty,oneof=monthly yearly"`
	IsActive     *bool           `json:"is_active"`
}

type ListPlanQuery struct {
	// IsActive hanya berlaku untuk platform admin; user lain selalu melihat plan aktif.
	IsActive     *bool   `form:"is_active"`
	BillingCycle *string `form:"billing_cycle" binding:"omitempty,oneof=monthly yearly"`
	Search       *string `form:"search"`
	PaginationParams
}

type PlanIDParam struct {
	PlanID string `uri:"planId" binding:"required,uuid"`
}

type PlanResponse struct {
	ID           string    `json:"id"`
	SolutionID   string    `json:"solution_id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Description  *string   `json:"description"`
	Price        string    `json:"price" example:"150000.00"`
	Currency     string    `json:"currency"`
	BillingCycle string    `json:"billing_cycle"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ReplaceBenefitsRequest — replace-all: daftar ini menjadi seluruh benefit plan / add-on.
// Array kosong menghapus semua benefit.
type ReplaceBenefitsRequest struct {
	Benefits []BenefitValueInput `json:"benefits" binding:"required,dive"`
}

type BenefitValueInput struct {
	BenefitID string          `json:"benefit_id" binding:"required,uuid"`
	Value     json.RawMessage `json:"value" binding:"required" swaggertype:"object"`
}

type AssignedBenefitResponse struct {
	BenefitID           string          `json:"benefit_id"`
	Code                string          `json:"code"`
	Name                string          `json:"name"`
	ValueType           string          `json:"value_type"`
	AggregationStrategy string          `json:"aggregation_strategy"`
	Unit                *string         `json:"unit"`
	Metering            string          `json:"metering"`
	Value               json.RawMessage `json:"value" swaggertype:"object"`
}
