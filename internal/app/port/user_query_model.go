package port

import (
	"context"

	"platform-service/internal/app/dto"
)

type UserListReadQuery struct {
	Status          *string
	IsPlatformAdmin *bool
	Search          *string // email / name
	dto.PaginationParams
}

type UserQueryReadModel interface {
	List(ctx context.Context, q UserListReadQuery) ([]dto.UserResponse, dto.Meta, error)
	// ListMemberships — semua keanggotaan company user (termasuk invited / inactive).
	ListMemberships(ctx context.Context, userID string) ([]dto.UserMembershipResponse, error)
}
