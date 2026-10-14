package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/permission/constant"
)

// PostgresPermissionQuery implements port.PermissionQueryReadModel.
type PostgresPermissionQuery struct {
	db *sql.DB
}

func NewPostgresPermissionQuery(db *sql.DB) *PostgresPermissionQuery {
	return &PostgresPermissionQuery{db: db}
}

var permissionSortable = map[string]string{"code": "code", "name": "name", "created_at": "created_at"}

func (r *PostgresPermissionQuery) List(ctx context.Context, q port.PermissionListReadQuery) ([]dto.PermissionResponse, dto.Meta, error) {
	limit, offset, page, sortColumn, sortType := resolvePaginationParams(q.PaginationParams, 50, 200, permissionSortable, "code", "asc")

	var w whereBuilder
	w.add(`solution_id = $%d`, q.SolutionID)
	if q.Status != nil {
		w.add(`status = $%d`, *q.Status)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		w.add(`(code ILIKE $%d OR name ILIKE $%d)`, likePattern(*q.Search), likePattern(*q.Search))
	}

	exec := execFromContext(ctx, r.db)
	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) FROM permissions `+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	rows, err := exec.QueryContext(ctx, fmt.Sprintf(`SELECT %s FROM permissions %s ORDER BY %s %s, id LIMIT %d OFFSET %d`,
		permissionColumns, w.sql(), sortColumn, sortType, limit, offset), w.args...)
	if err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	defer rows.Close()

	items := make([]dto.PermissionResponse, 0, limit)
	for rows.Next() {
		var it dto.PermissionResponse
		if err := rows.Scan(&it.ID, &it.SolutionID, &it.Code, &it.Name, &it.Description, &it.Status, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	return items, dto.NewMeta(page, limit, total), nil
}
