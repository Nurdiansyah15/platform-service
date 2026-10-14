package persistence

import (
	"context"
	"database/sql"

	"platform-service/internal/domain/entitlement/constant"
	"platform-service/internal/domain/entitlement/entity"
	domainerr "platform-service/internal/domain/errors"
)

// PostgresSnapshotReader implements repository.SnapshotReader (domain entitlement).
type PostgresSnapshotReader struct {
	db *sql.DB
}

func NewPostgresSnapshotReader(db *sql.DB) *PostgresSnapshotReader {
	return &PostgresSnapshotReader{db: db}
}

// EffectiveSnapshots: snapshot plan + snapshot add-on yang add-on-nya masih active.
// Waktu pasang add-on = updated_at subscription_addons (berubah saat diaktifkan ulang).
func (r *PostgresSnapshotReader) EffectiveSnapshots(ctx context.Context, subscriptionID string) ([]entity.SnapshotValue, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx, `
		SELECT s.benefit_id, s.value, s.subscription_addon_id IS NOT NULL, COALESCE(sa.quantity, 1),
			COALESCE(sa.updated_at, s.created_at)
		FROM subscription_benefit_snapshots s
		LEFT JOIN subscription_addons sa ON sa.id = s.subscription_addon_id
		WHERE s.subscription_id = $1
		  AND (s.subscription_addon_id IS NULL OR sa.status = 'active')`, subscriptionID)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeEntitlementQueryFailed, err)
	}
	defer rows.Close()

	var out []entity.SnapshotValue
	for rows.Next() {
		var (
			v     entity.SnapshotValue
			value []byte
		)
		if err := rows.Scan(&v.BenefitID, &value, &v.FromAddon, &v.Quantity, &v.AttachedAt); err != nil {
			return nil, domainerr.Wrap(constant.CodeEntitlementQueryFailed, err)
		}
		v.Value = value
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeEntitlementQueryFailed, err)
	}
	return out, nil
}
