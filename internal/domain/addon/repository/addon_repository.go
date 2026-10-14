package repository

import (
	"context"

	"platform-service/internal/domain/addon/entity"
)

type AddonRepository interface {
	// Save insert/update. Code bentrok dalam solution → CodeAddonCodeDuplicate.
	Save(ctx context.Context, a *entity.Addon) error
	// FindByID mengabaikan add-on yang sudah di-soft delete.
	FindByID(ctx context.Context, id string) (*entity.Addon, error)
}

type AddonBenefitRepository interface {
	// ReplaceAll mengganti seluruh benefit add-on dengan items (panggil di dalam transaksi).
	ReplaceAll(ctx context.Context, addonID string, items []entity.AddonBenefit) error
}
