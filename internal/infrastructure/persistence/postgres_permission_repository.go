package persistence

import (
	"context"
	"database/sql"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/permission/constant"
	"platform-service/internal/domain/permission/entity"
	"platform-service/internal/domain/permission/valueobject"
)

// PostgresPermissionRepository implements repository.PermissionRepository.
type PostgresPermissionRepository struct {
	db *sql.DB
}

func NewPostgresPermissionRepository(db *sql.DB) *PostgresPermissionRepository {
	return &PostgresPermissionRepository{db: db}
}

const permissionColumns = `id, solution_id, code, name, description, status, created_at, updated_at`

func (r *PostgresPermissionRepository) Save(ctx context.Context, p *entity.Permission) error {
	const q = `
		INSERT INTO permissions (id, solution_id, code, name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			status = EXCLUDED.status
		RETURNING created_at, updated_at`

	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		p.ID, p.SolutionID, p.Code.Value(), p.Name, p.Description, p.Status.String(), p.CreatedAt, p.UpdatedAt,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodePermissionCodeDuplicate)
		}
		return domainerr.Wrap(constant.CodePermissionPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresPermissionRepository) ListBySolution(ctx context.Context, solutionID string) ([]*entity.Permission, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx,
		`SELECT `+permissionColumns+` FROM permissions WHERE solution_id = $1 ORDER BY code`, solutionID)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	defer rows.Close()

	var out []*entity.Permission
	for rows.Next() {
		var (
			p            entity.Permission
			code, status string
		)
		if err := rows.Scan(&p.ID, &p.SolutionID, &code, &p.Name, &p.Description, &status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
		}
		if p.Code, err = valueobject.NewPermissionCode(code); err != nil {
			return nil, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
		}
		p.Status = valueobject.PermissionStatus(status)
		out = append(out, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	return out, nil
}
