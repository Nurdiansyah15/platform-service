package port

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
)

type TenantListReadQuery struct {
	// MemberUserID — bila diisi, hanya tenant tempat user ini member active.
	MemberUserID *string
	Status       *string
	Search       *string
	dto.PaginationParams
}

type TenantReadItem struct {
	ID        string
	Name      string
	Slug      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TenantMemberListReadQuery struct {
	TenantID string
	Status   *string
	Search   *string // email / nama user
	dto.PaginationParams
}

type TenantMemberReadItem struct {
	UserID   string
	Email    string
	Name     string
	Status   string
	JoinedAt time.Time
}

type TenantQueryReadModel interface {
	ListTenants(ctx context.Context, q TenantListReadQuery) ([]TenantReadItem, dto.Meta, error)
	ListMembers(ctx context.Context, q TenantMemberListReadQuery) ([]TenantMemberReadItem, dto.Meta, error)
}
