package port

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
)

type OrgUnitListReadQuery struct {
	// ParentID = company_id pemilik branch.
	ParentID string
	Status   *string
	Search   *string // code / name
	dto.PaginationParams
}

type OrgUnitReadItem struct {
	ID        string
	ParentID  string
	Name      string
	Code      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrganizationQueryReadModel interface {
	ListBranches(ctx context.Context, q OrgUnitListReadQuery) ([]OrgUnitReadItem, dto.Meta, error)
}
