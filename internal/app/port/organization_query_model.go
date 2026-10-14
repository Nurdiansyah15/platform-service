package port

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
)

type OrgUnitListReadQuery struct {
	// ParentID = tenant_id (company) atau company_id (branch).
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
	ListCompanies(ctx context.Context, q OrgUnitListReadQuery) ([]OrgUnitReadItem, dto.Meta, error)
	ListBranches(ctx context.Context, q OrgUnitListReadQuery) ([]OrgUnitReadItem, dto.Meta, error)
}
