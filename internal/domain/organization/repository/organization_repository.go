package repository

import (
	"context"

	"platform-service/internal/domain/organization/entity"
)

type CompanyRepository interface {
	// Save insert/update. Code bentrok dalam tenant → CodeCompanyCodeDuplicate.
	Save(ctx context.Context, c *entity.Company) error
	// FindByID mengabaikan company yang sudah di-soft delete.
	FindByID(ctx context.Context, id string) (*entity.Company, error)
}

type BranchRepository interface {
	// Save insert/update. Code bentrok dalam company → CodeBranchCodeDuplicate.
	Save(ctx context.Context, b *entity.Branch) error
}
