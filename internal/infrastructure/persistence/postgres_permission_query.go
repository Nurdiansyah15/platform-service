package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
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
			// Group beserta seluruh turunannya.
			w.add(`p.group_id IN (
				WITH RECURSIVE sub AS (
					SELECT id FROM permission_groups WHERE solution_id = p.solution_id AND code = $%d
					UNION ALL
					SELECT c.id FROM permission_groups c JOIN sub ON c.parent_id = sub.id
				) SELECT id FROM sub)`, *q.Group)
		}
	}

	const from = `FROM permissions p LEFT JOIN permission_groups g ON g.id = p.group_id `
	exec := execFromContext(ctx, r.db)
	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) `+from+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	rows, err := exec.QueryContext(ctx, fmt.Sprintf(`
		SELECT p.id, p.solution_id, p.code, p.name, p.description, p.status, p.source, p.created_at, p.updated_at,
			g.code, g.name, g.sort_order, `+groupParentsSQL+`
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
			&it.CreatedAt, &it.UpdatedAt, &group.code, &group.name, &group.sortOrder, &group.parents); err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
		}
		var err error
		if it.Group, err = group.ref(); err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	return items, dto.NewMeta(page, limit, total), nil
}

// groupParentsSQL — leluhur group alias `g` (root → induk langsung) sebagai array JSONB
// {code, name, sort_order}. Dibatasi 10 langkah agar data siklus tidak membuat query berputar.
const groupParentsSQL = `COALESCE((
	WITH RECURSIVE anc AS (
		SELECT a.id, a.parent_id, a.code, a.name, a.sort_order, 1 AS lvl FROM permission_groups a WHERE a.id = g.parent_id
		UNION ALL
		SELECT a.id, a.parent_id, a.code, a.name, a.sort_order, anc.lvl + 1
		FROM permission_groups a JOIN anc ON a.id = anc.parent_id WHERE anc.lvl < 10
	)
	SELECT jsonb_agg(jsonb_build_object('code', code, 'name', name, 'sort_order', sort_order) ORDER BY lvl DESC) FROM anc
), '[]'::jsonb)`

// nullableGroupRef — kolom permission_groups hasil LEFT JOIN (+ groupParentsSQL).
type nullableGroupRef struct {
	code, name sql.NullString
	sortOrder  sql.NullInt64
	parents    []byte
}

func (g nullableGroupRef) ref() (*dto.PermissionGroupRef, error) {
	if !g.code.Valid {
		return nil, nil
	}
	ref := &dto.PermissionGroupRef{Code: g.code.String, Name: g.name.String, SortOrder: int(g.sortOrder.Int64),
		Parents: []dto.PermissionGroupAncestor{}}
	if len(g.parents) > 0 {
		if err := json.Unmarshal(g.parents, &ref.Parents); err != nil {
			return nil, err
		}
	}
	return ref, nil
}
