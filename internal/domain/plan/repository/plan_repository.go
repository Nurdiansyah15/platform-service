package repository

import (
	"context"

	"platform-service/internal/domain/plan/entity"
)

type PlanRepository interface {
	// Save insert/update. Code bentrok dalam solution → CodePlanCodeDuplicate.
	Save(ctx context.Context, p *entity.Plan) error
	// FindByID mengabaikan plan yang sudah di-soft delete.
	FindByID(ctx context.Context, id string) (*entity.Plan, error)
}

type PlanBenefitRepository interface {
	// ReplaceAll mengganti seluruh benefit plan dengan items (panggil di dalam transaksi).
	ReplaceAll(ctx context.Context, planID string, items []entity.PlanBenefit) error
}
