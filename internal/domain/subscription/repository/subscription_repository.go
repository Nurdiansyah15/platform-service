package repository

import (
	"context"

	"platform-service/internal/domain/subscription/entity"
)

type SubscriptionRepository interface {
	// Save insert/update. Subscription live kedua untuk company+solution → CodeSubscriptionAlreadyLive.
	Save(ctx context.Context, s *entity.Subscription) error
	FindByID(ctx context.Context, id string) (*entity.Subscription, error)
	// HasUsable — company punya subscription trialing / active untuk solution.
	HasUsable(ctx context.Context, companyID, solutionID string) (bool, error)
	// FindUsable mengembalikan subscription trialing / active company untuk solution.
	FindUsable(ctx context.Context, companyID, solutionID string) (*entity.Subscription, error)
}

type SubscriptionAddonRepository interface {
	Save(ctx context.Context, a *entity.SubscriptionAddon) error
	// Find mencari baris (subscription_id, addon_id) apa pun statusnya.
	Find(ctx context.Context, subscriptionID, addonID string) (*entity.SubscriptionAddon, error)
}

type SnapshotRepository interface {
	// ReplacePlanSnapshots mengganti snapshot asal plan (subscription_addon_id NULL).
	ReplacePlanSnapshots(ctx context.Context, subscriptionID string, rows []entity.BenefitSnapshot) error
	// ReplaceAddonSnapshots mengganti snapshot milik satu subscription add-on.
	ReplaceAddonSnapshots(ctx context.Context, subscriptionAddonID string, rows []entity.BenefitSnapshot) error
}
