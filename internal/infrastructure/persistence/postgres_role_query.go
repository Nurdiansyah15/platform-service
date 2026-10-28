package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/role/constant"
)

// PostgresRoleQuery implements port.RoleQueryReadModel.
type PostgresRoleQuery struct {
	db *sql.DB
}

func NewPostgresRoleQuery(db *sql.DB) *PostgresRoleQuery {
	return &PostgresRoleQuery{db: db}
}

var roleSortable = map[string]string{"code": "code", "name": "name", "created_at": "created_at"}

func (r *PostgresRoleQuery) ListRoles(ctx context.Context, q port.RoleListReadQuery) ([]port.RoleReadItem, dto.Meta, error) {
	limit, offset, page, sortColumn, sortType := resolvePaginationParams(q.PaginationParams, 50, 200, roleSortable, "code", "asc")

	var w whereBuilder
	w.add(`solution_id = $%d`, q.SolutionID)
	w.add(`deleted_at IS NULL`)
	if q.CompanyID == nil {
		w.add(`company_id IS NULL`)
	} else {
		w.add(`(company_id IS NULL OR company_id = $%d)`, *q.CompanyID)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		w.add(`(code ILIKE $%d OR name ILIKE $%d)`, likePattern(*q.Search), likePattern(*q.Search))
	}

	exec := execFromContext(ctx, r.db)
	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) FROM roles `+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
	}
	// Role global tampil lebih dulu, lalu custom role company.
	rows, err := exec.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, solution_id, company_id, code, name, description, is_system, created_at, updated_at
		FROM roles %s ORDER BY (company_id IS NOT NULL), %s %s, id LIMIT %d OFFSET %d`,
		w.sql(), sortColumn, sortType, limit, offset), w.args...)
	if err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
	}
	defer rows.Close()

	items := make([]port.RoleReadItem, 0, limit)
	for rows.Next() {
		var it port.RoleReadItem
		if err := rows.Scan(&it.ID, &it.SolutionID, &it.CompanyID, &it.Code, &it.Name, &it.Description, &it.IsSystem,
			&it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
	}
	return items, dto.NewMeta(page, limit, total), nil
}

func (r *PostgresRoleQuery) ListRolePermissions(ctx context.Context, roleID string) ([]port.RolePermissionReadItem, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx, `
		SELECT p.id, p.code, p.name, p.status
		FROM role_permissions rp JOIN permissions p ON p.id = rp.permission_id
		WHERE rp.role_id = $1 ORDER BY p.code`, roleID)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
	}
	defer rows.Close()
	items := []port.RolePermissionReadItem{}
	for rows.Next() {
		var it port.RolePermissionReadItem
		if err := rows.Scan(&it.ID, &it.Code, &it.Name, &it.Status); err != nil {
			return nil, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
	}
	return items, nil
}
