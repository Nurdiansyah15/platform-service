package port

import (
	"context"
	"encoding/json"
	"time"

	"platform-service/internal/app/dto"
)

type PlanListReadQuery struct {
	SolutionID   string
	IsActive     *bool
	BillingCycle *string
	Search       *string // code / name
	dto.PaginationParams
}

type PlanReadItem struct {
	ID           string
	SolutionID   string
	Code         string
	Name         string
	Description  *string
	Price        string
	Currency     string
	BillingCycle string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AssignedBenefitReadItem adalah nilai benefit pada plan / add-on beserta info benefit-nya.
type AssignedBenefitReadItem struct {
	BenefitID           string
	Code                string
	Name                string
	ValueType           string
	AggregationStrategy string
	Unit                *string
	Metering            string
	Value               json.RawMessage
}

type PlanQueryReadModel interface {
	ListPlans(ctx context.Context, q PlanListReadQuery) ([]PlanReadItem, dto.Meta, error)
	// ListPlanBenefits mengembalikan semua benefit plan, urut code (tanpa pagination — jumlahnya kecil).
	ListPlanBenefits(ctx context.Context, planID string) ([]AssignedBenefitReadItem, error)
}
