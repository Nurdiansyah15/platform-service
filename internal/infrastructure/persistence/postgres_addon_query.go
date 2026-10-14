package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/domain/addon/constant"
	domainerr "platform-service/internal/domain/errors"
)

// PostgresAddonQuery implements port.AddonQueryReadModel.
type PostgresAddonQuery struct {
	db *sql.DB
}

func NewPostgresAddonQuery(db *sql.DB) *PostgresAddonQuery {
	return &PostgresAddonQuery{db: db}
}

// Kolom sort di-qualify — lihat catatan planSortable.
var addonSortable = map[string]string{"code": "addons.code", "name": "addons.name", "price": "addons.price", "created_at": "addons.created_at"}

func (r *PostgresAddonQuery) ListAddons(ctx context.Context, q port.AddonListReadQuery) ([]port.AddonReadItem, dto.Meta, error) {
	limit, offset, page, sortColumn, sortType := resolvePaginationParams(q.PaginationParams, 20, 100, addonSortable, "addons.price", "asc")

	var w whereBuilder
	w.add(`solution_id = $%d`, q.SolutionID)
	w.add(`deleted_at IS NULL`)
	if q.IsActive != nil {
		w.add(`is_active = $%d`, *q.IsActive)
	}
	if q.BillingCycle != nil {
		w.add(`billing_cycle = $%d`, *q.BillingCycle)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		w.add(`(code ILIKE $%d OR name ILIKE $%d)`, likePattern(*q.Search), likePattern(*q.Search))
	}

	exec := execFromContext(ctx, r.db)
	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) FROM addons `+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeAddonQueryFailed, err)
	}

	rows, err := exec.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, solution_id, code, name, description, price::text, currency, billing_cycle, is_active, created_at, updated_at
		FROM addons %s ORDER BY %s %s, addons.id LIMIT %d OFFSET %d`, w.sql(), sortColumn, sortType, limit, offset), w.args...)
	if err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeAddonQueryFailed, err)
	}
	defer rows.Close()

	items := make([]port.AddonReadItem, 0, limit)
	for rows.Next() {
		var it port.AddonReadItem
		if err := rows.Scan(&it.ID, &it.SolutionID, &it.Code, &it.Name, &it.Description, &it.Price, &it.Currency,
			&it.BillingCycle, &it.IsActive, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodeAddonQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeAddonQueryFailed, err)
	}
	return items, dto.NewMeta(page, limit, total), nil
}

func (r *PostgresAddonQuery) ListAddonBenefits(ctx context.Context, addonID string) ([]port.AssignedBenefitReadItem, error) {
	return listAssignedBenefits(ctx, execFromContext(ctx, r.db), `
		SELECT b.id, b.code, b.name, b.value_type, b.aggregation_strategy, b.unit, b.metering, ab.value
		FROM addon_benefits ab JOIN benefits b ON b.id = ab.benefit_id
		WHERE ab.addon_id = $1 ORDER BY b.code`, addonID, constant.CodeAddonQueryFailed)
}
