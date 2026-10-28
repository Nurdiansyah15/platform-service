package repository

import (
	"context"

	"platform-service/internal/domain/organization/entity"
)

type CompanyRepository interface {
	// Save insert/update. Slug bentrok → CodeCompanySlugDuplicate.
	Save(ctx context.Context, c *entity.Company) error
	FindByID(ctx context.Context, id string) (*entity.Company, error)
}

type CompanyMemberRepository interface {
	// Save insert/update keanggotaan (upsert per company_id + user_id).
	Save(ctx context.Context, m *entity.CompanyMember) error
	Find(ctx context.Context, companyID, userID string) (*entity.CompanyMember, error)
}

type BranchRepository interface {
	// Save insert/update. Code bentrok dalam company → CodeBranchCodeDuplicate.
	Save(ctx context.Context, b *entity.Branch) error
	// FindByID mengabaikan branch yang sudah di-soft delete.
	FindByID(ctx context.Context, id string) (*entity.Branch, error)
}
