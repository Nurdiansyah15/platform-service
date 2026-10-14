package repository

import (
	"context"

	"platform-service/internal/domain/benefit/entity"
)

type BenefitRepository interface {
	// Save insert/update. Code bentrok dalam solution → CodeBenefitCodeDuplicate.
	Save(ctx context.Context, b *entity.Benefit) error
	FindByID(ctx context.Context, id string) (*entity.Benefit, error)
	FindByCode(ctx context.Context, solutionID, code string) (*entity.Benefit, error)
	// ListBySolution mengembalikan semua benefit solution, urut code.
	ListBySolution(ctx context.Context, solutionID string) ([]*entity.Benefit, error)
	// FindByIDs mengembalikan benefit yang ditemukan (urutan tidak dijamin; ID tak dikenal diabaikan).
	FindByIDs(ctx context.Context, ids []string) ([]*entity.Benefit, error)
}
