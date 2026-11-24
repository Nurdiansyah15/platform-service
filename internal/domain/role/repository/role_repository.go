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
	// Current mengembalikan permission ID → scope_type yang terpasang pada role.
	Current(ctx context.Context, roleID string) (map[string]string, error)
	// ScopeTypes mengembalikan scope_type unik yang dipakai permission role (terurut).
	ScopeTypes(ctx context.Context, roleID string) ([]string, error)
	// ReplaceAll mengganti seluruh permission role (panggil di dalam transaksi).
	ReplaceAll(ctx context.Context, roleID, solutionID string, grants []entity.PermissionGrant) error
}
