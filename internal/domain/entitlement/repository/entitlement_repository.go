package repository

import (
	"context"

	"platform-service/internal/domain/entitlement/entity"
)

// SnapshotReader membaca snapshot efektif subscription: snapshot plan + snapshot
// add-on yang masih active (beserta quantity & waktu pasang add-on).
type SnapshotReader interface {
	EffectiveSnapshots(ctx context.Context, subscriptionID string) ([]entity.SnapshotValue, error)
}
