package repository

import (
	"context"

	"platform-service/internal/domain/permission/entity"
)

type PermissionRepository interface {
	// Save insert/update permission.
	Save(ctx context.Context, p *entity.Permission) error
	// ListBySolution mengembalikan seluruh permission solution (semua status).
	ListBySolution(ctx context.Context, solutionID string) ([]*entity.Permission, error)
}
