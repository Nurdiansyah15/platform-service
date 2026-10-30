package persistence

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/solution/constant"
	"platform-service/internal/domain/solution/entity"
	"platform-service/internal/domain/solution/valueobject"
)

// PostgresSolutionRepository implements repository.SolutionRepository.
type PostgresSolutionRepository struct {
	db *sql.DB
	// typeMap dipakai untuk scan kolom TEXT[] (database/sql tidak mendukung array).
	typeMap *pgtype.Map
}

func NewPostgresSolutionRepository(db *sql.DB) *PostgresSolutionRepository {
	return &PostgresSolutionRepository{db: db, typeMap: pgtype.NewMap()}
}

const solutionColumns = `id, code, name, description, status, permission_catalog_version,
	launch_url, icon_url, redirect_uris, created_at, updated_at`

func (r *PostgresSolutionRepository) Save(ctx context.Context, s *entity.Solution) error {
	const q = `
		INSERT INTO solutions (id, code, name, description, status, permission_catalog_version,
			launch_url, icon_url, redirect_uris, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			status = EXCLUDED.status,
			permission_catalog_version = EXCLUDED.permission_catalog_version,
			launch_url = EXCLUDED.launch_url,
			icon_url = EXCLUDED.icon_url,
			redirect_uris = EXCLUDED.redirect_uris
		RETURNING created_at, updated_at`

	redirects := s.RedirectURIs
	if redirects == nil {
		redirects = []string{}
	}
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		s.ID, s.Code.Value(), s.Name, s.Description, s.Status.String(), s.PermissionCatalogVersion,
		s.LaunchURL, s.IconURL, redirects, s.CreatedAt, s.UpdatedAt,
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
	return r.findOne(ctx, `SELECT `+solutionColumns+` FROM solutions WHERE id = $1`, id)
}

func (r *PostgresSolutionRepository) FindByCode(ctx context.Context, code string) (*entity.Solution, error) {
	return r.findOne(ctx, `SELECT `+solutionColumns+` FROM solutions WHERE code = $1`, code)
}

func (r *PostgresSolutionRepository) FindByIDForUpdate(ctx context.Context, id string) (*entity.Solution, error) {
	return r.findOne(ctx, `SELECT `+solutionColumns+` FROM solutions WHERE id = $1 FOR UPDATE`, id)
}

func (r *PostgresSolutionRepository) ListLaunchable(ctx context.Context) ([]*entity.Solution, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx,
		`SELECT `+solutionColumns+` FROM solutions WHERE status = 'active' AND launch_url IS NOT NULL ORDER BY name, id`)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeSolutionQueryFailed, err)
	}
	defer rows.Close()

	var out []*entity.Solution
	for rows.Next() {
		s, err := r.scanSolution(rows)
		if err != nil {
			return nil, domainerr.Wrap(constant.CodeSolutionQueryFailed, err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeSolutionQueryFailed, err)
	}
	return out, nil
}

func (r *PostgresSolutionRepository) findOne(ctx context.Context, q string, args ...any) (*entity.Solution, error) {
	s, err := r.scanSolution(execFromContext(ctx, r.db).QueryRowContext(ctx, q, args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeSolutionNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeSolutionQueryFailed, err)
	}
	return s, nil
}

func (r *PostgresSolutionRepository) scanSolution(row interface{ Scan(dest ...any) error }) (*entity.Solution, error) {
	var (
		s            entity.Solution
		code, status string
	)
	if err := row.Scan(&s.ID, &code, &s.Name, &s.Description, &status, &s.PermissionCatalogVersion,
		&s.LaunchURL, &s.IconURL, r.typeMap.SQLScanner(&s.RedirectURIs), &s.CreatedAt, &s.UpdatedAt); err != nil {
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
