package port

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
)

type RoleListReadQuery struct {
	SolutionID string
	// TenantID nil → hanya role global; terisi → role global + custom role tenant tersebut.
	TenantID *string
	Search   *string // code / name
	dto.PaginationParams
}

type RoleReadItem struct {
	ID          string
	SolutionID  string
	TenantID    *string
	Code        string
	Name        string
	Description *string
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
}
