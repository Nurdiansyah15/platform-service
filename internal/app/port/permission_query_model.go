package port

import (
	"context"

	"platform-service/internal/app/dto"
)

type PermissionListReadQuery struct {
	SolutionID string
	Status     *string
	Search     *string // code / name
	dto.PaginationParams
}

type PermissionQueryReadModel interface {
	List(ctx context.Context, q PermissionListReadQuery) ([]dto.PermissionResponse, dto.Meta, error)
}
