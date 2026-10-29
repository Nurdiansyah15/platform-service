package port

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
)

type RoleListReadQuery struct {
	SolutionID string
	// CompanyID nil → hanya role global; terisi → role global + custom role company tersebut.
	CompanyID *string
	Search    *string // code / name
	Kind      *string // global | custom
	dto.PaginationParams
}

type RoleReadItem struct {
	ID          string
	SolutionID  string
	CompanyID   *string
	Code        string
	Name        string
	Description *string
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time

	PermissionCount int
	MemberCount     int
}

// RoleStats — jumlah permission & pemegang role (MemberCount dibatasi ke CompanyID bila diisi).
type RoleStats struct {
	PermissionCount int
	MemberCount     int
}

type RolePermissionReadItem struct {
	ID     string
	Code   string
	Name   string
	Status string
}

type RoleQueryReadModel interface {
	ListRoles(ctx context.Context, q RoleListReadQuery) ([]RoleReadItem, dto.Meta, error)
	ListRolePermissions(ctx context.Context, roleID string) ([]RolePermissionReadItem, error)
	Stats(ctx context.Context, roleID string, companyID *string) (RoleStats, error)
	// ListHolders — assignment role di company beserta user & nama scope.
	ListHolders(ctx context.Context, roleID, companyID string) ([]dto.RoleHolderResponse, error)
}
