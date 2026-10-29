package port

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
)

type CompanyListReadQuery struct {
	// MemberUserID — bila diisi, hanya company tempat user ini member active.
	MemberUserID *string
	Status       *string
	Search       *string
	dto.PaginationParams
}

type CompanyReadItem struct {
	ID        string
	Name      string
	Slug      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
	// MemberCount — jumlah member active.
	MemberCount int
}

type CompanyMemberListReadQuery struct {
	CompanyID string
	Status    *string
	Search    *string // email / nama user
	dto.PaginationParams
}

type CompanyMemberReadItem struct {
	UserID   string
	Email    string
	Name     string
	Status   string
	JoinedAt time.Time
	// RoleCount — jumlah role assignment member di company.
	RoleCount int
}

type CompanyQueryReadModel interface {
	ListCompanies(ctx context.Context, q CompanyListReadQuery) ([]CompanyReadItem, dto.Meta, error)
	ListMembers(ctx context.Context, q CompanyMemberListReadQuery) ([]CompanyMemberReadItem, dto.Meta, error)
	// SearchMemberCandidates — user active (email / nama cocok) yang belum member active / invited.
	SearchMemberCandidates(ctx context.Context, companyID, search string, limit int) ([]dto.MemberCandidateResponse, error)
	Summary(ctx context.Context, companyID string) (*dto.CompanySummaryResponse, error)
}
