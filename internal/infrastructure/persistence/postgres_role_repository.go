package persistence

import (
	"context"
	"database/sql"
	"errors"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/role/constant"
	"platform-service/internal/domain/role/entity"
	"platform-service/internal/domain/role/valueobject"
)

// PostgresRoleRepository implements repository.RoleRepository.
type PostgresRoleRepository struct {
	db *sql.DB
}

func NewPostgresRoleRepository(db *sql.DB) *PostgresRoleRepository {
	return &PostgresRoleRepository{db: db}
}

func (r *PostgresRoleRepository) Save(ctx context.Context, role *entity.Role) error {
	const q = `
		INSERT INTO roles (id, solution_id, company_id, code, name, description, is_system, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			code = EXCLUDED.code,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			is_system = EXCLUDED.is_system,
			deleted_at = EXCLUDED.deleted_at
		RETURNING created_at, updated_at`

	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		role.ID, role.SolutionID, role.CompanyID, role.Code.Value(), role.Name, role.Description, role.IsSystem,
		role.CreatedAt, role.UpdatedAt, role.DeletedAt,
	).Scan(&role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodeRoleCodeDuplicate)
		}
		return domainerr.Wrap(constant.CodeRolePersistenceFailed, err)
	}
	return nil
}

func (r *PostgresRoleRepository) FindByID(ctx context.Context, id string) (*entity.Role, error) {
	return r.findOne(ctx, `WHERE id = $1 AND deleted_at IS NULL`, id)
}

func (r *PostgresRoleRepository) FindGlobalByCode(ctx context.Context, solutionID, code string) (*entity.Role, error) {
	return r.findOne(ctx, `WHERE solution_id = $1 AND company_id IS NULL AND code = $2 AND deleted_at IS NULL`, solutionID, code)
}

func (r *PostgresRoleRepository) findOne(ctx context.Context, where string, args ...any) (*entity.Role, error) {
	var (
		role entity.Role
		code string
	)
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT id, solution_id, company_id, code, name, description, is_system, created_at, updated_at, deleted_at
		FROM roles `+where, args...,
	).Scan(&role.ID, &role.SolutionID, &role.CompanyID, &code, &role.Name, &role.Description, &role.IsSystem,
		&role.CreatedAt, &role.UpdatedAt, &role.DeletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeRoleNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
	}
	if role.Code, err = valueobject.NewRoleCode(code); err != nil {
		return nil, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
	}
	return &role, nil
}

// PostgresRolePermissionRepository implements repository.RolePermissionRepository.
type PostgresRolePermissionRepository struct {
	db *sql.DB
}

func NewPostgresRolePermissionRepository(db *sql.DB) *PostgresRolePermissionRepository {
	return &PostgresRolePermissionRepository{db: db}
}

func (r *PostgresRolePermissionRepository) Current(ctx context.Context, roleID string) (map[string]string, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx,
		`SELECT permission_id, scope_type FROM role_permissions WHERE role_id = $1`, roleID)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, scope string
		if err := rows.Scan(&id, &scope); err != nil {
			return nil, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
		}
		out[id] = scope
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
	}
	return out, nil
}

func (r *PostgresRolePermissionRepository) ScopeTypes(ctx context.Context, roleID string) ([]string, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx,
		`SELECT DISTINCT scope_type FROM role_permissions WHERE role_id = $1 ORDER BY scope_type`, roleID)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var scope string
		if err := rows.Scan(&scope); err != nil {
			return nil, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
		}
		out = append(out, scope)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeRoleQueryFailed, err)
	}
	return out, nil
}

func (r *PostgresRolePermissionRepository) ReplaceAll(ctx context.Context, roleID, solutionID string, grants []entity.PermissionGrant) error {
	exec := execFromContext(ctx, r.db)
	if _, err := exec.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
		return domainerr.Wrap(constant.CodeRolePersistenceFailed, err)
	}
	if len(grants) == 0 {
		return nil
	}
	ids := make([]string, len(grants))
	scopes := make([]string, len(grants))
	for i, g := range grants {
		ids[i], scopes[i] = g.PermissionID, g.ScopeType
	}
	_, err := exec.ExecContext(ctx, `
		INSERT INTO role_permissions (role_id, permission_id, solution_id, scope_type)
		SELECT $1, p.id, $2, p.scope FROM UNNEST($3::uuid[], $4::text[]) AS p(id, scope)`, roleID, solutionID, ids, scopes)
	if err != nil {
		if isForeignKeyViolation(err) {
			return domainerr.New(constant.CodeRolePermissionUnknown)
		}
		if isCheckViolation(err) {
			return domainerr.New(constant.CodeRolePermissionScopeInvalid)
		}
		return domainerr.Wrap(constant.CodeRolePersistenceFailed, err)
	}
	return nil
}
