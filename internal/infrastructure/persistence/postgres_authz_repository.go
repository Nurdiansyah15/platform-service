package persistence

import (
	"context"
	"database/sql"

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
		INSERT INTO user_role_assignments (id, tenant_id, user_id, role_id, scope_type, scope_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		a.ID, a.TenantID, a.UserID, a.RoleID, a.Scope.Type.String(), a.Scope.ID, a.CreatedAt)
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

// PostgresGrantReader implements repository.GrantReader.
type PostgresGrantReader struct {
	db *sql.DB
}

func NewPostgresGrantReader(db *sql.DB) *PostgresGrantReader {
	return &PostgresGrantReader{db: db}
}

// Grants: assignment user di tenant → role solution (belum dihapus; global atau custom milik
// tenant) → permission role (termasuk deprecated yang masih terpasang).
func (r *PostgresGrantReader) Grants(ctx context.Context, tenantID, userID, solutionID string) ([]entity.Grant, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx, `
		SELECT r.code, p.code, ura.scope_type, ura.scope_id
		FROM user_role_assignments ura
		JOIN roles r ON r.id = ura.role_id
		JOIN role_permissions rp ON rp.role_id = r.id
		JOIN permissions p ON p.id = rp.permission_id
		WHERE ura.tenant_id = $1 AND ura.user_id = $2 AND r.solution_id = $3
		  AND r.deleted_at IS NULL AND (r.tenant_id IS NULL OR r.tenant_id = ura.tenant_id)`,
		tenantID, userID, solutionID)
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
