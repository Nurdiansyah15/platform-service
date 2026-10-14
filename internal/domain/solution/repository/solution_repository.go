package repository

import (
	"context"

	"platform-service/internal/domain/solution/entity"
)

type SolutionRepository interface {
	Save(ctx context.Context, s *entity.Solution) error
	FindByID(ctx context.Context, id string) (*entity.Solution, error)
	FindByCode(ctx context.Context, code string) (*entity.Solution, error)
}
