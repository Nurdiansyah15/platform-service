// Package service berisi domain service subscription.
package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"platform-service/internal/domain/subscription/entity"
)

// BenefitValue adalah nilai satu benefit pada plan / add-on master.
type BenefitValue struct {
	BenefitID string
	Value     json.RawMessage
}

// BenefitValueSource membaca nilai benefit master (plan_benefits / addon_benefits).
type BenefitValueSource interface {
	PlanBenefitValues(ctx context.Context, planID string) ([]BenefitValue, error)
	AddonBenefitValues(ctx context.Context, addonID string) ([]BenefitValue, error)
}

// SnapshotBuilder membekukan nilai benefit ke snapshot subscription (spec §2.5):
//   - create subscription / change plan → salin plan_benefits (subscription_addon_id NULL)
//   - tambah add-on → salin addon_benefits (subscription_addon_id terisi)
type SnapshotBuilder struct {
	source BenefitValueSource
}

func NewSnapshotBuilder(source BenefitValueSource) *SnapshotBuilder {
	return &SnapshotBuilder{source: source}
}

func (b *SnapshotBuilder) ForPlan(ctx context.Context, s *entity.Subscription, now time.Time) ([]entity.BenefitSnapshot, error) {
	values, err := b.source.PlanBenefitValues(ctx, s.PlanID)
	if err != nil {
		return nil, err
	}
	return toSnapshots(s.ID, nil, values, now), nil
}

func (b *SnapshotBuilder) ForAddon(ctx context.Context, a *entity.SubscriptionAddon, now time.Time) ([]entity.BenefitSnapshot, error) {
	values, err := b.source.AddonBenefitValues(ctx, a.AddonID)
	if err != nil {
		return nil, err
	}
	id := a.ID
	return toSnapshots(a.SubscriptionID, &id, values, now), nil
}

func toSnapshots(subscriptionID string, subscriptionAddonID *string, values []BenefitValue, now time.Time) []entity.BenefitSnapshot {
	out := make([]entity.BenefitSnapshot, 0, len(values))
	for _, v := range values {
		out = append(out, entity.BenefitSnapshot{
			ID:                  uuid.NewString(),
			SubscriptionID:      subscriptionID,
			SubscriptionAddonID: subscriptionAddonID,
			BenefitID:           v.BenefitID,
			Value:               v.Value,
			CreatedAt:           now.UTC(),
		})
	}
	return out
}
