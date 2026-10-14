package repository

import (
	"context"

	"platform-service/internal/domain/benefit/entity"
)

type BenefitRepository interface {
	// Save insert/update. Code bentrok dalam solution → CodeBenefitCodeDuplicate.
	Save(ctx context.Context, b *entity.Benefit) error
	FindByID(ctx context.Context, id string) (*entity.Benefit, error)
}
