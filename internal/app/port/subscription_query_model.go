package port

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
)

type SubscriptionListReadQuery struct {
	TenantID   string
	SolutionID *string // dipaksa untuk token service (hanya solution miliknya)
	Status     *string
	dto.PaginationParams
}

type SubscriptionReadItem struct {
	ID                 string
	TenantID           string
	SolutionID         string
	SolutionCode       string
	PlanID             string
	PlanCode           string
	PlanName           string
	Status             string
	BillingCycle       string
	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time
	PriceAtPurchase    string
	Currency           string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type SubscriptionAddonReadItem struct {
	ID                  string
	SubscriptionID      string
	AddonID             string
	AddonCode           string
	AddonName           string
	Quantity            int
	UnitPriceAtPurchase string
	Currency            string
	Status              string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// SelectableTenantReadItem — tenant yang boleh dipilih user untuk solution sesi:
// user member active, tenant active, subscription trialing / active.
type SelectableTenantReadItem struct {
	ID     string
	Name   string
	Slug   string
	Status string
}

type SubscriptionQueryReadModel interface {
	ListSubscriptions(ctx context.Context, q SubscriptionListReadQuery) ([]SubscriptionReadItem, dto.Meta, error)
	GetSubscription(ctx context.Context, id string) (*SubscriptionReadItem, error)
	ListSubscriptionAddons(ctx context.Context, subscriptionID string) ([]SubscriptionAddonReadItem, error)
	GetSubscriptionAddon(ctx context.Context, id string) (*SubscriptionAddonReadItem, error)
	ListSelectableTenants(ctx context.Context, userID, solutionID string) ([]SelectableTenantReadItem, error)
}
