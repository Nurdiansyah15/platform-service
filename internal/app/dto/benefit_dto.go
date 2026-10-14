package dto

import "time"

type CreateBenefitRequest struct {
	Code                string  `json:"code" binding:"required,max=100"`
	Name                string  `json:"name" binding:"required,max=255"`
	Description         *string `json:"description"`
	ValueType           string  `json:"value_type" binding:"required,oneof=boolean integer decimal string json"`
	AggregationStrategy string  `json:"aggregation_strategy" binding:"required,oneof=sum max min override any_true"`
	Unit                *string `json:"unit" binding:"omitempty,max=50"`
	Metering            string  `json:"metering" binding:"omitempty,oneof=none counter gauge"`
	UsageReset          string  `json:"usage_reset" binding:"omitempty,oneof=never billing_period monthly"`
}

type ListBenefitQuery struct {
	ValueType *string `form:"value_type" binding:"omitempty,oneof=boolean integer decimal string json"`
	Metering  *string `form:"metering" binding:"omitempty,oneof=none counter gauge"`
	Search    *string `form:"search"`
	PaginationParams
}

type BenefitIDParam struct {
	BenefitID string `uri:"benefitId" binding:"required,uuid"`
}

type BenefitResponse struct {
	ID                  string    `json:"id"`
	SolutionID          string    `json:"solution_id"`
	Code                string    `json:"code"`
	Name                string    `json:"name"`
	Description         *string   `json:"description"`
	ValueType           string    `json:"value_type"`
	AggregationStrategy string    `json:"aggregation_strategy"`
	Unit                *string   `json:"unit"`
	Metering            string    `json:"metering"`
	UsageReset          string    `json:"usage_reset"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
