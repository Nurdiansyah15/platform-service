package repository

import (
	"context"

	"platform-service/internal/domain/authz/entity"
)

type RoleAssignmentRepository interface {
	// Create menyimpan assignment. Kombinasi user+role+scope sudah ada → CodeAssignmentDuplicate.
	Create(ctx context.Context, a *entity.RoleAssignment) error
}

// GrantReader membaca seluruh permission user di company untuk satu solution
// (assignment → role yang belum dihapus → permission).
type GrantReader interface {
	Grants(ctx context.Context, companyID, userID, solutionID string) ([]entity.Grant, error)
}
