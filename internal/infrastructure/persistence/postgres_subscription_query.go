package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/subscription/constant"
	"platform-service/internal/domain/subscription/valueobject"
)

// PostgresSubscriptionQuery implements port.SubscriptionQueryReadModel.
type PostgresSubscriptionQuery struct {
	db *sql.DB
}

func NewPostgresSubscriptionQuery(db *sql.DB) *PostgresSubscriptionQuery {
	return &PostgresSubscriptionQuery{db: db}
}

var subscriptionSortable = map[string]string{"created_at": "s.created_at", "status": "s.status", "current_period_end": "s.current_period_end"}

const subscriptionSelect = `
	SELECT s.id, s.tenant_id, s.solution_id, sol.code, s.plan_id, p.code, p.name, s.status, s.billing_cycle,
		s.current_period_start, s.current_period_end, s.price_at_purchase::text, s.currency, s.created_at, s.updated_at
	FROM subscriptions s
	JOIN solutions sol ON sol.id = s.solution_id
	JOIN plans p ON p.id = s.plan_id`

func scanSubscriptionItem(row interface{ Scan(dest ...any) error }) (port.SubscriptionReadItem, error) {
	var it port.SubscriptionReadItem
	err := row.Scan(&it.ID, &it.TenantID, &it.SolutionID, &it.SolutionCode, &it.PlanID, &it.PlanCode, &it.PlanName,
		&it.Status, &it.BillingCycle, &it.CurrentPeriodStart, &it.CurrentPeriodEnd, &it.PriceAtPurchase, &it.Currency,
		&it.CreatedAt, &it.UpdatedAt)
	return it, err
}

func (r *PostgresSubscriptionQuery) ListSubscriptions(ctx context.Context, q port.SubscriptionListReadQuery) ([]port.SubscriptionReadItem, dto.Meta, error) {
	limit, offset, page, sortColumn, sortType := resolvePaginationParams(q.PaginationParams, 20, 100, subscriptionSortable, "s.created_at", "desc")

	var w whereBuilder
	w.add(`s.tenant_id = $%d`, q.TenantID)
	if q.SolutionID != nil {
		w.add(`s.solution_id = $%d`, *q.SolutionID)
	}
	if q.Status != nil {
		w.add(`s.status = $%d`, *q.Status)
	}

	exec := execFromContext(ctx, r.db)
	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) FROM subscriptions s `+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}

	rows, err := exec.QueryContext(ctx, fmt.Sprintf(`%s %s ORDER BY %s %s, s.id LIMIT %d OFFSET %d`,
		subscriptionSelect, w.sql(), sortColumn, sortType, limit, offset), w.args...)
	if err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	defer rows.Close()

	items := make([]port.SubscriptionReadItem, 0, limit)
	for rows.Next() {
		it, err := scanSubscriptionItem(rows)
		if err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	return items, dto.NewMeta(page, limit, total), nil
}

func (r *PostgresSubscriptionQuery) GetSubscription(ctx context.Context, id string) (*port.SubscriptionReadItem, error) {
	it, err := scanSubscriptionItem(execFromContext(ctx, r.db).QueryRowContext(ctx, subscriptionSelect+` WHERE s.id = $1`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeSubscriptionNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	return &it, nil
}

const subscriptionAddonSelect = `
	SELECT sa.id, sa.subscription_id, sa.addon_id, a.code, a.name, sa.quantity, sa.unit_price_at_purchase::text,
		sa.currency, sa.status, sa.created_at, sa.updated_at
	FROM subscription_addons sa JOIN addons a ON a.id = sa.addon_id`

func scanSubscriptionAddonItem(row interface{ Scan(dest ...any) error }) (port.SubscriptionAddonReadItem, error) {
	var it port.SubscriptionAddonReadItem
	err := row.Scan(&it.ID, &it.SubscriptionID, &it.AddonID, &it.AddonCode, &it.AddonName, &it.Quantity,
		&it.UnitPriceAtPurchase, &it.Currency, &it.Status, &it.CreatedAt, &it.UpdatedAt)
	return it, err
}

func (r *PostgresSubscriptionQuery) ListSubscriptionAddons(ctx context.Context, subscriptionID string) ([]port.SubscriptionAddonReadItem, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx,
		subscriptionAddonSelect+` WHERE sa.subscription_id = $1 ORDER BY sa.status, a.code`, subscriptionID)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	defer rows.Close()
	items := []port.SubscriptionAddonReadItem{}
	for rows.Next() {
		it, err := scanSubscriptionAddonItem(rows)
		if err != nil {
			return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	return items, nil
}

func (r *PostgresSubscriptionQuery) GetSubscriptionAddon(ctx context.Context, id string) (*port.SubscriptionAddonReadItem, error) {
	it, err := scanSubscriptionAddonItem(execFromContext(ctx, r.db).QueryRowContext(ctx, subscriptionAddonSelect+` WHERE sa.id = $1`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeSubscriptionAddonNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	return &it, nil
}

func (r *PostgresSubscriptionQuery) ListSelectableTenants(ctx context.Context, userID, solutionID string) ([]port.SelectableTenantReadItem, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx, `
		SELECT t.id, t.name, t.slug, t.status
		FROM tenants t
		JOIN tenant_members m ON m.tenant_id = t.id AND m.user_id = $1 AND m.status = 'active'
		WHERE t.status = 'active'
		  AND EXISTS (SELECT 1 FROM subscriptions s
		              WHERE s.tenant_id = t.id AND s.solution_id = $2 AND s.status = ANY($3::text[]))
		ORDER BY t.name, t.id`, userID, solutionID, valueobject.UsableStatuses)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	defer rows.Close()
	items := []port.SelectableTenantReadItem{}
	for rows.Next() {
		var it port.SelectableTenantReadItem
		if err := rows.Scan(&it.ID, &it.Name, &it.Slug, &it.Status); err != nil {
			return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	return items, nil
}
