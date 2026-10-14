package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/organization/constant"
)

// PostgresOrganizationQuery implements port.OrganizationQueryReadModel.
type PostgresOrganizationQuery struct {
	db *sql.DB
}

func NewPostgresOrganizationQuery(db *sql.DB) *PostgresOrganizationQuery {
	return &PostgresOrganizationQuery{db: db}
}

var orgUnitSortable = map[string]string{"code": "code", "name": "name", "created_at": "created_at"}

func (r *PostgresOrganizationQuery) ListCompanies(ctx context.Context, q port.OrgUnitListReadQuery) ([]port.OrgUnitReadItem, dto.Meta, error) {
	return r.list(ctx, "companies", "tenant_id", q)
}

func (r *PostgresOrganizationQuery) ListBranches(ctx context.Context, q port.OrgUnitListReadQuery) ([]port.OrgUnitReadItem, dto.Meta, error) {
	return r.list(ctx, "branches", "company_id", q)
}

// list — table & parentColumn berasal dari konstanta di atas, bukan input user.
func (r *PostgresOrganizationQuery) list(ctx context.Context, table, parentColumn string, q port.OrgUnitListReadQuery) ([]port.OrgUnitReadItem, dto.Meta, error) {
	limit, offset, page, sortColumn, sortType := resolvePaginationParams(q.PaginationParams, 20, 100, orgUnitSortable, "code", "asc")

	var w whereBuilder
	w.add(parentColumn+` = $%d`, q.ParentID)
	w.add(`deleted_at IS NULL`)
	if q.Status != nil {
		w.add(`status = $%d`, *q.Status)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		w.add(`(code ILIKE $%d OR name ILIKE $%d)`, likePattern(*q.Search), likePattern(*q.Search))
	}

	exec := execFromContext(ctx, r.db)
	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table+` `+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeOrgQueryFailed, err)
	}

	rows, err := exec.QueryContext(ctx, fmt.Sprintf(
		`SELECT id, %s, name, code, status, created_at, updated_at FROM %s %s ORDER BY %s %s, id LIMIT %d OFFSET %d`,
		parentColumn, table, w.sql(), sortColumn, sortType, limit, offset), w.args...)
	if err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeOrgQueryFailed, err)
	}
	defer rows.Close()

	items := make([]port.OrgUnitReadItem, 0, limit)
	for rows.Next() {
		var it port.OrgUnitReadItem
		if err := rows.Scan(&it.ID, &it.ParentID, &it.Name, &it.Code, &it.Status, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodeOrgQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeOrgQueryFailed, err)
	}
	return items, dto.NewMeta(page, limit, total), nil
}
