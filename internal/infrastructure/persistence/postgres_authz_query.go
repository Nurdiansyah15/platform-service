package persistence

import (
	"context"
	"database/sql"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/domain/authz/constant"
	domainerr "platform-service/internal/domain/errors"
)

// PostgresRoleAssignmentQuery implements port.RoleAssignmentQueryReadModel.
type PostgresRoleAssignmentQuery struct {
	db *sql.DB
}

func NewPostgresRoleAssignmentQuery(db *sql.DB) *PostgresRoleAssignmentQuery {
	return &PostgresRoleAssignmentQuery{db: db}
}

func (r *PostgresRoleAssignmentQuery) ListForMember(ctx context.Context, q port.RoleAssignmentListReadQuery) ([]dto.RoleAssignmentDetailResponse, error) {
	var w whereBuilder
	w.add(`ura.company_id = $%d`, q.CompanyID)
	w.add(`ura.user_id = $%d`, q.UserID)
	if q.SolutionID != nil {
		w.add(`r.solution_id = $%d`, *q.SolutionID)
	}
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx, `
		SELECT ura.id, ura.company_id, ura.user_id, r.id, r.code, r.name, r.company_id IS NULL,
			s.id, s.code, ura.scope_type, ura.scope_id,
			COALESCE(CASE ura.scope_type WHEN 'branch' THEN b.name ELSE c.name END, ''),
			ura.created_at
		FROM user_role_assignments ura
		JOIN roles r ON r.id = ura.role_id
		JOIN solutions s ON s.id = r.solution_id
		LEFT JOIN branches b ON ura.scope_type = 'branch' AND b.id = ura.scope_id
		LEFT JOIN companies c ON ura.scope_type = 'company' AND c.id = ura.scope_id
		`+w.sql()+`
		ORDER BY s.code, r.code, ura.created_at, ura.id`, w.args...)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeAuthzQueryFailed, err)
	}
	defer rows.Close()

	items := []dto.RoleAssignmentDetailResponse{}
	for rows.Next() {
		var it dto.RoleAssignmentDetailResponse
		if err := rows.Scan(&it.ID, &it.CompanyID, &it.UserID, &it.RoleID, &it.RoleCode, &it.RoleName, &it.RoleIsGlobal,
			&it.SolutionID, &it.SolutionCode, &it.ScopeType, &it.ScopeID, &it.ScopeName, &it.CreatedAt); err != nil {
			return nil, domainerr.Wrap(constant.CodeAuthzQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeAuthzQueryFailed, err)
	}
	return items, nil
}
