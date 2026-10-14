package port

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
)

type AddonListReadQuery struct {
	SolutionID   string
	IsActive     *bool
	BillingCycle *string
	Search       *string // code / name
	dto.PaginationParams
}

type AddonReadItem struct {
	ID           string
	SolutionID   string
	Code         string
	Name         string
	Description  *string
	Price        string
	Currency     string
	BillingCycle string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AddonQueryReadModel interface {
	ListAddons(ctx context.Context, q AddonListReadQuery) ([]AddonReadItem, dto.Meta, error)
	// ListAddonBenefits mengembalikan semua benefit add-on, urut code.
	ListAddonBenefits(ctx context.Context, addonID string) ([]AssignedBenefitReadItem, error)
}
