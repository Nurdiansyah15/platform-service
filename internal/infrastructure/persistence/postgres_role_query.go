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
	if q.Kind != nil {
		if *q.Kind == "custom" {
			w.add(`company_id IS NOT NULL`)
		} else {
			w.add(`company_id IS NULL`)
		}
	}

	exec := execFromContext(ctx, r.db)
	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) FROM roles `+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
	}
	// Jumlah pemegang: dalam konteks company hanya assignment di company tersebut.
	memberFilter := ""
	if q.CompanyID != nil {
		w.args = append(w.args, *q.CompanyID)
		memberFilter = fmt.Sprintf(" AND ura.company_id = $%d", len(w.args))
	}
	// Role global tampil lebih dulu, lalu custom role company.
	rows, err := exec.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, solution_id, company_id, code, name, description, is_system, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = roles.id),
			(SELECT COUNT(DISTINCT ura.user_id) FROM user_role_assignments ura WHERE ura.role_id = roles.id%s)
		FROM roles %s ORDER BY (company_id IS NOT NULL), %s %s, id LIMIT %d OFFSET %d`,
		memberFilter, w.sql(), sortColumn, sortType, limit, offset), w.args...)
	if err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
	}
	defer rows.Close()

	items := make([]port.RoleReadItem, 0, limit)
	for rows.Next() {
		var it port.RoleReadItem
		if err := rows.Scan(&it.ID, &it.SolutionID, &it.CompanyID, &it.Code, &it.Name, &it.Description, &it.IsSystem,
			&it.CreatedAt, &it.UpdatedAt, &it.PermissionCount, &it.MemberCount); err != nil {
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

func (r *PostgresRoleQuery) Stats(ctx context.Context, roleID string, companyID *string) (port.RoleStats, error) {
	var st port.RoleStats
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM role_permissions WHERE role_id = $1),
			(SELECT COUNT(DISTINCT user_id) FROM user_role_assignments
			 WHERE role_id = $1 AND ($2::uuid IS NULL OR company_id = $2::uuid))`, roleID, companyID,
	).Scan(&st.PermissionCount, &st.MemberCount)
	if err != nil {
		return port.RoleStats{}, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
	}
	return st, nil
}

func (r *PostgresRoleQuery) ListHolders(ctx context.Context, roleID, companyID string) ([]dto.RoleHolderResponse, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx, `
		SELECT ura.id, u.id, u.name, u.email, ura.scope_type, ura.scope_id,
			COALESCE(CASE ura.scope_type WHEN 'branch' THEN b.name ELSE c.name END, ''),
			ura.created_at
		FROM user_role_assignments ura
		JOIN users u ON u.id = ura.user_id
		LEFT JOIN branches b ON ura.scope_type = 'branch' AND b.id = ura.scope_id
		LEFT JOIN companies c ON ura.scope_type = 'company' AND c.id = ura.scope_id
		WHERE ura.role_id = $1 AND ura.company_id = $2
		ORDER BY u.name, u.email, ura.created_at`, roleID, companyID)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
	}
	defer rows.Close()
	items := []dto.RoleHolderResponse{}
	for rows.Next() {
		var it dto.RoleHolderResponse
		if err := rows.Scan(&it.AssignmentID, &it.UserID, &it.Name, &it.Email, &it.ScopeType, &it.ScopeID, &it.ScopeName, &it.CreatedAt); err != nil {
			return nil, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
	}
	return items, nil
}
