package port

import (
	"context"

	"platform-service/internal/app/dto"
)

type BenefitListReadQuery struct {
	SolutionID string
	ValueType  *string
	Metering   *string
	Search     *string // code / name
	dto.PaginationParams
}

type BenefitQueryReadModel interface {
	// List mengembalikan dto.BenefitResponse langsung karena bentuknya identik dengan baris tabel.
	List(ctx context.Context, q BenefitListReadQuery) ([]dto.BenefitResponse, dto.Meta, error)
}
