package persistence

import (
	"context"
	"database/sql"
	"encoding/json"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/domain/authz/constant"
	domainerr "platform-service/internal/domain/errors"
)

// assignmentScopesSQL — nilai scope assignment sebagai array JSON [{type, type_name, value, name}]
// (urut type). Nama branch di-join; scope custom memakai label atau value. Alias wajib:
// ura (user_role_assignments), r (roles).
const assignmentScopesSQL = `COALESCE((
		SELECT jsonb_agg(jsonb_build_object(
			'type', kv.key,
			'type_name', CASE kv.key WHEN 'branch' THEN 'Branch' ELSE COALESCE(sst.name, kv.key) END,
			'value', kv.value,
			'name', COALESCE(CASE kv.key WHEN 'branch' THEN b.name END, ura.scope_labels ->> kv.key, kv.value)
		) ORDER BY kv.key)
		FROM jsonb_each_text(ura.scope_values) kv
		LEFT JOIN branches b ON kv.key = 'branch' AND b.id::text = kv.value
		LEFT JOIN solution_scope_types sst ON sst.solution_id = r.solution_id AND sst.code = kv.key
	), '[]'::jsonb)`

// scanAssignmentScopes mengisi scope_values, scope_labels, dan scopes dari kolom JSONB.
func scanAssignmentScopes(values, labels, scopes []byte) (map[string]string, map[string]string, []dto.AssignmentScope, error) {
	v, err := parseJSONObject(values)
	if err != nil {
		return nil, nil, nil, err
	}
	l, err := parseJSONObject(labels)
	if err != nil {
		return nil, nil, nil, err
	}
	out := []dto.AssignmentScope{}
	if len(scopes) > 0 {
		if err := json.Unmarshal(scopes, &out); err != nil {
			return nil, nil, nil, err
		}
	}
	return v, l, out, nil
}

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
			s.id, s.code, ura.scope_values, ura.scope_labels, `+assignmentScopesSQL+`,
			COALESCE((SELECT jsonb_agg(DISTINCT rp.scope_type) FROM role_permissions rp WHERE rp.role_id = r.id), '[]'::jsonb),
			ura.created_at
		FROM user_role_assignments ura
		JOIN roles r ON r.id = ura.role_id
		JOIN solutions s ON s.id = r.solution_id
		`+w.sql()+`
		ORDER BY s.code, r.code, ura.created_at, ura.id`, w.args...)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeAuthzQueryFailed, err)
	}
	defer rows.Close()

	items := []dto.RoleAssignmentDetailResponse{}
	for rows.Next() {
		var (
			it                     dto.RoleAssignmentDetailResponse
			values, labels, scopes []byte
			roleScopes             []byte
		)
		if err := rows.Scan(&it.ID, &it.CompanyID, &it.UserID, &it.RoleID, &it.RoleCode, &it.RoleName, &it.RoleIsGlobal,
			&it.SolutionID, &it.SolutionCode, &values, &labels, &scopes, &roleScopes, &it.CreatedAt); err != nil {
			return nil, domainerr.Wrap(constant.CodeAuthzQueryFailed, err)
		}
		var err error
		if it.ScopeValues, it.ScopeLabels, it.Scopes, err = scanAssignmentScopes(values, labels, scopes); err != nil {
			return nil, domainerr.Wrap(constant.CodeAuthzQueryFailed, err)
		}
		it.RoleScopeTypes = []string{}
		if err := json.Unmarshal(roleScopes, &it.RoleScopeTypes); err != nil {
			return nil, domainerr.Wrap(constant.CodeAuthzQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeAuthzQueryFailed, err)
	}
	return items, nil
}
