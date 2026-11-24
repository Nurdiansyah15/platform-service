package persistence

import (
	"context"
	"database/sql"
	"errors"

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

const permissionColumns = `id, solution_id, code, name, description, status, source, group_id, created_at, updated_at`

func (r *PostgresPermissionRepository) Save(ctx context.Context, p *entity.Permission) error {
	const q = `
		INSERT INTO permissions (` + permissionColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			status = EXCLUDED.status,
			group_id = EXCLUDED.group_id
		RETURNING created_at, updated_at`

	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		p.ID, p.SolutionID, p.Code.Value(), p.Name, p.Description, p.Status.String(), p.Source.String(), p.GroupID,
		p.CreatedAt, p.UpdatedAt,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodePermissionCodeDuplicate)
		}
		if isForeignKeyViolation(err) {
			return domainerr.New(constant.CodeGroupUnknown)
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
		p, err := scanPermission(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	return out, nil
}

func (r *PostgresPermissionRepository) FindByID(ctx context.Context, id string) (*entity.Permission, error) {
	p, err := scanPermission(execFromContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT `+permissionColumns+` FROM permissions WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodePermissionNotFound)
		}
		return nil, err
	}
	return p, nil
}

func scanPermission(row interface{ Scan(dest ...any) error }) (*entity.Permission, error) {
	var (
		p                    entity.Permission
		code, status, source string
	)
	if err := row.Scan(&p.ID, &p.SolutionID, &code, &p.Name, &p.Description, &status, &source, &p.GroupID,
		&p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	var err error
	if p.Code, err = valueobject.NewPermissionCode(code); err != nil {
		return nil, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	p.Status = valueobject.PermissionStatus(status)
	p.Source = valueobject.PermissionSource(source)
	return &p, nil
}
