package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/permission/constant"
)

// PostgresPermissionQuery implements port.PermissionQueryReadModel.
type PostgresPermissionQuery struct {
	db      *sql.DB
	typeMap *pgtype.Map
}

func NewPostgresPermissionQuery(db *sql.DB) *PostgresPermissionQuery {
	return &PostgresPermissionQuery{db: db, typeMap: pgtype.NewMap()}
}

var permissionSortable = map[string]string{"code": "p.code", "name": "p.name", "created_at": "p.created_at"}

func (r *PostgresPermissionQuery) List(ctx context.Context, q port.PermissionListReadQuery) ([]dto.PermissionResponse, dto.Meta, error) {
	limit, offset, page, sortColumn, sortType := resolvePaginationParams(q.PaginationParams, 50, 200, permissionSortable, "p.code", "asc")

	var w whereBuilder
	w.add(`p.solution_id = $%d`, q.SolutionID)
	if q.Status != nil {
		w.add(`p.status = $%d`, *q.Status)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		w.add(`(p.code ILIKE $%d OR p.name ILIKE $%d)`, likePattern(*q.Search), likePattern(*q.Search))
	}
	if q.Source != nil {
		w.add(`p.source = $%d`, *q.Source)
	}
	if q.Group != nil {
		if *q.Group == "-" {
			w.add(`p.group_id IS NULL`)
		} else {
			w.add(`g.code = $%d`, *q.Group)
		}
	}

	const from = `FROM permissions p LEFT JOIN permission_groups g ON g.id = p.group_id `
	exec := execFromContext(ctx, r.db)
	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) `+from+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	rows, err := exec.QueryContext(ctx, fmt.Sprintf(`
		SELECT p.id, p.solution_id, p.code, p.name, p.description, p.status, p.source, p.scope_types, p.created_at, p.updated_at,
			g.code, g.name, g.sort_order
		%s %s ORDER BY %s %s, p.id LIMIT %d OFFSET %d`,
		from, w.sql(), sortColumn, sortType, limit, offset), w.args...)
	if err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	defer rows.Close()

	items := make([]dto.PermissionResponse, 0, limit)
	for rows.Next() {
		var (
			it    dto.PermissionResponse
			group nullableGroupRef
		)
		if err := rows.Scan(&it.ID, &it.SolutionID, &it.Code, &it.Name, &it.Description, &it.Status, &it.Source,
			r.typeMap.SQLScanner(&it.ScopeTypes), &it.CreatedAt, &it.UpdatedAt, &group.code, &group.name, &group.sortOrder); err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
		}
		if it.ScopeTypes == nil {
			it.ScopeTypes = []string{}
		}
		it.Group = group.ref()
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	return items, dto.NewMeta(page, limit, total), nil
}

// nullableGroupRef — kolom permission_groups hasil LEFT JOIN.
type nullableGroupRef struct {
	code, name sql.NullString
	sortOrder  sql.NullInt64
}

func (g nullableGroupRef) ref() *dto.PermissionGroupRef {
	if !g.code.Valid {
		return nil
	}
	return &dto.PermissionGroupRef{Code: g.code.String, Name: g.name.String, SortOrder: int(g.sortOrder.Int64)}
}
