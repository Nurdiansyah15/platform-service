package repository

import (
	"context"

	"platform-service/internal/domain/role/entity"
)

type RoleRepository interface {
	// Save insert/update. Code bentrok (global per solution / custom per company+solution) → CodeRoleCodeDuplicate.
	Save(ctx context.Context, r *entity.Role) error
	// FindByID mengabaikan role yang sudah di-soft delete.
	FindByID(ctx context.Context, id string) (*entity.Role, error)
	// FindGlobalByCode — role global (company_id NULL) solution dengan code tersebut.
	FindGlobalByCode(ctx context.Context, solutionID, code string) (*entity.Role, error)
}

type RolePermissionRepository interface {
	// PermissionIDs mengembalikan ID permission yang terpasang pada role.
	PermissionIDs(ctx context.Context, roleID string) (map[string]bool, error)
	// ReplaceAll mengganti seluruh permission role (panggil di dalam transaksi).
	ReplaceAll(ctx context.Context, roleID, solutionID string, permissionIDs []string) error
}
