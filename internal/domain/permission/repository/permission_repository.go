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
	// FindByID memuat permission. Tidak ada → CodePermissionNotFound.
	FindByID(ctx context.Context, id string) (*entity.Permission, error)
}

type PermissionGroupRepository interface {
	// Save insert/update group.
	Save(ctx context.Context, g *entity.Group) error
	// ListBySolution mengembalikan seluruh group solution (semua status), urut sort_order.
	ListBySolution(ctx context.Context, solutionID string) ([]*entity.Group, error)
	// FindByID memuat group. Tidak ada → CodeGroupNotFound.
	FindByID(ctx context.Context, id string) (*entity.Group, error)
}

type ScopeTypeRepository interface {
	// Save insert/update scope type.
	Save(ctx context.Context, s *entity.ScopeType) error
	// ListBySolution mengembalikan seluruh scope type custom solution (semua status), urut sort_order.
	ListBySolution(ctx context.Context, solutionID string) ([]*entity.ScopeType, error)
	// FindByCode memuat scope type custom (semua status). Tidak ada → CodeScopeTypeNotFound.
	FindByCode(ctx context.Context, solutionID, code string) (*entity.ScopeType, error)
	// FindByID memuat scope type custom. Tidak ada → CodeScopeTypeNotFound.
	FindByID(ctx context.Context, id string) (*entity.ScopeType, error)
}
