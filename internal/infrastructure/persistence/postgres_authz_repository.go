package persistence

import (
	"context"
	"database/sql"
	"errors"

	"platform-service/internal/domain/authz/constant"
	"platform-service/internal/domain/authz/entity"
	"platform-service/internal/domain/authz/valueobject"
	domainerr "platform-service/internal/domain/errors"
)

// PostgresRoleAssignmentRepository implements repository.RoleAssignmentRepository.
type PostgresRoleAssignmentRepository struct {
	db *sql.DB
}

func NewPostgresRoleAssignmentRepository(db *sql.DB) *PostgresRoleAssignmentRepository {
	return &PostgresRoleAssignmentRepository{db: db}
}

func (r *PostgresRoleAssignmentRepository) Create(ctx context.Context, a *entity.RoleAssignment) error {
	_, err := execFromContext(ctx, r.db).ExecContext(ctx, `
		INSERT INTO user_role_assignments (id, company_id, user_id, role_id, scope_type, scope_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		a.ID, a.CompanyID, a.UserID, a.RoleID, a.Scope.Type.String(), a.Scope.ID, a.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodeAssignmentDuplicate)
		}
		if isForeignKeyViolation(err) {
			return domainerr.New(constant.CodeAssignmentNotMember)
		}
		return domainerr.Wrap(constant.CodeAuthzPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresRoleAssignmentRepository) FindByID(ctx context.Context, id string) (*entity.RoleAssignment, error) {
	var (
		a         entity.RoleAssignment
		scopeType string
	)
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT ura.id, ura.company_id, ura.user_id, ura.role_id, ura.scope_type, ura.scope_id, ura.created_at,
			r.solution_id, r.code
		FROM user_role_assignments ura
		JOIN roles r ON r.id = ura.role_id
		WHERE ura.id = $1`, id,
	).Scan(&a.ID, &a.CompanyID, &a.UserID, &a.RoleID, &scopeType, &a.Scope.ID, &a.CreatedAt, &a.Role.SolutionID, &a.Role.Code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeAssignmentNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeAuthzQueryFailed, err)
	}
	if a.Scope.Type, err = valueobject.NewScopeType(scopeType); err != nil {
		return nil, domainerr.Wrap(constant.CodeAuthzQueryFailed, err)
	}
	return &a, nil
}

func (r *PostgresRoleAssignmentRepository) Delete(ctx context.Context, id string) error {
	if _, err := execFromContext(ctx, r.db).ExecContext(ctx, `DELETE FROM user_role_assignments WHERE id = $1`, id); err != nil {
		return domainerr.Wrap(constant.CodeAuthzPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresRoleAssignmentRepository) DeleteByScope(ctx context.Context, scopeType, scopeID string) error {
	_, err := execFromContext(ctx, r.db).ExecContext(ctx,
		`DELETE FROM user_role_assignments WHERE scope_type = $1 AND scope_id = $2`, scopeType, scopeID)
	if err != nil {
		return domainerr.Wrap(constant.CodeAuthzPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresRoleAssignmentRepository) DeleteByRole(ctx context.Context, roleID string) error {
	_, err := execFromContext(ctx, r.db).ExecContext(ctx, `DELETE FROM user_role_assignments WHERE role_id = $1`, roleID)
	if err != nil {
		return domainerr.Wrap(constant.CodeAuthzPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresRoleAssignmentRepository) DeleteByMember(ctx context.Context, companyID, userID string) error {
	_, err := execFromContext(ctx, r.db).ExecContext(ctx,
		`DELETE FROM user_role_assignments WHERE company_id = $1 AND user_id = $2`, companyID, userID)
	if err != nil {
		return domainerr.Wrap(constant.CodeAuthzPersistenceFailed, err)
	}
	return nil
}

// PostgresGrantReader implements repository.GrantReader.
type PostgresGrantReader struct {
	db *sql.DB
}

func NewPostgresGrantReader(db *sql.DB) *PostgresGrantReader {
	return &PostgresGrantReader{db: db}
}

// Grants: assignment user di company → role solution (belum dihapus; global atau custom milik
// company) → permission role (termasuk deprecated yang masih terpasang).
func (r *PostgresGrantReader) Grants(ctx context.Context, companyID, userID, solutionID string) ([]entity.Grant, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx, `
		SELECT r.code, p.code, ura.scope_type, ura.scope_id
		FROM user_role_assignments ura
		JOIN roles r ON r.id = ura.role_id
		JOIN role_permissions rp ON rp.role_id = r.id
		JOIN permissions p ON p.id = rp.permission_id
		WHERE ura.company_id = $1 AND ura.user_id = $2 AND r.solution_id = $3
		  AND r.deleted_at IS NULL AND (r.company_id IS NULL OR r.company_id = ura.company_id)`,
		companyID, userID, solutionID)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeAuthzQueryFailed, err)
	}
	defer rows.Close()

	var out []entity.Grant
	for rows.Next() {
		var (
			g         entity.Grant
			scopeType string
		)
		if err := rows.Scan(&g.RoleCode, &g.PermissionCode, &scopeType, &g.Scope.ID); err != nil {
			return nil, domainerr.Wrap(constant.CodeAuthzQueryFailed, err)
		}
		g.Scope.Type = valueobject.ScopeType(scopeType)
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeAuthzQueryFailed, err)
	}
	return out, nil
}
