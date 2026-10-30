package port

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
)

type SolutionListReadQuery struct {
	Status *string // nil → semua status
	Search *string // cocokkan code / name (case-insensitive)
	dto.PaginationParams
}

type SolutionReadItem struct {
	ID                       string
	Code                     string
	Name                     string
	Description              *string
	Status                   string
	PermissionCatalogVersion *string
	LaunchURL                *string
	IconURL                  *string
	RedirectURIs             []string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type SolutionQueryReadModel interface {
	List(ctx context.Context, q SolutionListReadQuery) ([]SolutionReadItem, dto.Meta, error)
}
