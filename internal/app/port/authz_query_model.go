package port

import (
	"context"

	"platform-service/internal/app/dto"
)

type RoleAssignmentListReadQuery struct {
	CompanyID  string
	UserID     string
	SolutionID *string
}

type RoleAssignmentQueryReadModel interface {
	// ListForMember — assignment user di company (termasuk role yang sudah soft delete).
	ListForMember(ctx context.Context, q RoleAssignmentListReadQuery) ([]dto.RoleAssignmentDetailResponse, error)
}
