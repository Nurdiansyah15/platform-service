package port

import (
	"context"

	"platform-service/internal/app/dto"
)

type SolutionClientListReadQuery struct {
	SolutionID string
	Status     *string
	Search     *string // name / client_id
	dto.PaginationParams
}

type SolutionClientQueryReadModel interface {
	List(ctx context.Context, q SolutionClientListReadQuery) ([]dto.SolutionClientResponse, dto.Meta, error)
}
