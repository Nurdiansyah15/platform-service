package persistence

import (
	"context"
	"database/sql"
	"errors"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/solution/constant"
	"platform-service/internal/domain/solution/entity"
	"platform-service/internal/domain/solution/valueobject"
)

// PostgresSolutionRepository implements repository.SolutionRepository.
type PostgresSolutionRepository struct {
	db *sql.DB
}

func NewPostgresSolutionRepository(db *sql.DB) *PostgresSolutionRepository {
	return &PostgresSolutionRepository{db: db}
}

const solutionColumns = `id, code, name, description, status, permission_catalog_version, created_at, updated_at`

func (r *PostgresSolutionRepository) Save(ctx context.Context, s *entity.Solution) error {
	const q = `
		INSERT INTO solutions (id, code, name, description, status, permission_catalog_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			status = EXCLUDED.status,
			permission_catalog_version = EXCLUDED.permission_catalog_version
		RETURNING created_at, updated_at`

	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		s.ID, s.Code.Value(), s.Name, s.Description, s.Status.String(), s.PermissionCatalogVersion, s.CreatedAt, s.UpdatedAt,
	).Scan(&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodeSolutionCodeDuplicate)
		}
		return domainerr.Wrap(constant.CodeSolutionPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresSolutionRepository) FindByID(ctx context.Context, id string) (*entity.Solution, error) {
	q := `SELECT ` + solutionColumns + ` FROM solutions WHERE id = $1`
	s, err := scanSolution(execFromContext(ctx, r.db).QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeSolutionNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeSolutionQueryFailed, err)
	}
	return s, nil
}

func scanSolution(row interface{ Scan(dest ...any) error }) (*entity.Solution, error) {
	var (
		s            entity.Solution
		code, status string
	)
	if err := row.Scan(&s.ID, &code, &s.Name, &s.Description, &status, &s.PermissionCatalogVersion, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	var err error
	if s.Code, err = valueobject.NewSolutionCode(code); err != nil {
		return nil, err
	}
	if s.Status, err = valueobject.NewSolutionStatus(status); err != nil {
		return nil, err
	}
	return &s, nil
}
