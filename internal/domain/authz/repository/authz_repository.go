package repository

import (
	"context"

	"platform-service/internal/domain/authz/entity"
)

type RoleAssignmentRepository interface {
	// Create menyimpan assignment. Kombinasi user+role+scope sudah ada → CodeAssignmentDuplicate.
	Create(ctx context.Context, a *entity.RoleAssignment) error
	// FindByID memuat assignment beserta ringkasan role-nya. Tidak ada → CodeAssignmentNotFound.
	FindByID(ctx context.Context, id string) (*entity.RoleAssignment, error)
	Delete(ctx context.Context, id string) error
	// DeleteByScope menghapus semua assignment pada scope tertentu (mis. branch dihapus).
	DeleteByScope(ctx context.Context, scopeType, scopeID string) error
	// DeleteByRole menghapus semua assignment role tertentu (role dihapus).
	DeleteByRole(ctx context.Context, roleID string) error
	// DeleteByMember menghapus semua assignment user di company (member dikeluarkan).
	DeleteByMember(ctx context.Context, companyID, userID string) error
}

// GrantReader membaca seluruh permission user di company untuk satu solution
// (assignment → role yang belum dihapus → permission).
type GrantReader interface {
	Grants(ctx context.Context, companyID, userID, solutionID string) ([]entity.Grant, error)
}
