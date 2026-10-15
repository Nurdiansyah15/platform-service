package persistence

import (
	"context"
	"database/sql"
	"errors"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/organization/constant"
	"platform-service/internal/domain/organization/entity"
	"platform-service/internal/domain/organization/valueobject"
)

// PostgresCompanyRepository implements repository.CompanyRepository.
type PostgresCompanyRepository struct {
	db *sql.DB
}

func NewPostgresCompanyRepository(db *sql.DB) *PostgresCompanyRepository {
	return &PostgresCompanyRepository{db: db}
}

func (r *PostgresCompanyRepository) Save(ctx context.Context, c *entity.Company) error {
	const q = `
		INSERT INTO companies (id, tenant_id, name, code, status, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			code = EXCLUDED.code,
			status = EXCLUDED.status,
			deleted_at = EXCLUDED.deleted_at
		RETURNING created_at, updated_at`

	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		c.ID, c.TenantID, c.Name, c.Code.Value(), c.Status.String(), c.CreatedAt, c.UpdatedAt, c.DeletedAt,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodeCompanyCodeDuplicate)
		}
		return domainerr.Wrap(constant.CodeOrgPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresCompanyRepository) FindByID(ctx context.Context, id string) (*entity.Company, error) {
	var (
		c            entity.Company
		code, status string
	)
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT id, tenant_id, name, code, status, created_at, updated_at, deleted_at
		FROM companies WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&c.ID, &c.TenantID, &c.Name, &code, &status, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeCompanyNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeOrgQueryFailed, err)
	}
	if c.Code, err = valueobject.NewOrgCode(code); err != nil {
		return nil, domainerr.Wrap(constant.CodeOrgQueryFailed, err)
	}
	if c.Status, err = valueobject.NewOrgStatus(status); err != nil {
		return nil, domainerr.Wrap(constant.CodeOrgQueryFailed, err)
	}
	return &c, nil
}

// PostgresBranchRepository implements repository.BranchRepository.
type PostgresBranchRepository struct {
	db *sql.DB
}

func NewPostgresBranchRepository(db *sql.DB) *PostgresBranchRepository {
	return &PostgresBranchRepository{db: db}
}

func (r *PostgresBranchRepository) Save(ctx context.Context, b *entity.Branch) error {
	const q = `
		INSERT INTO branches (id, company_id, name, code, status, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			code = EXCLUDED.code,
			status = EXCLUDED.status,
			deleted_at = EXCLUDED.deleted_at
		RETURNING created_at, updated_at`

	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		b.ID, b.CompanyID, b.Name, b.Code.Value(), b.Status.String(), b.CreatedAt, b.UpdatedAt, b.DeletedAt,
	).Scan(&b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodeBranchCodeDuplicate)
		}
		return domainerr.Wrap(constant.CodeOrgPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresBranchRepository) FindByID(ctx context.Context, id string) (*entity.Branch, error) {
	var (
		b            entity.Branch
		code, status string
	)
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT id, company_id, name, code, status, created_at, updated_at, deleted_at
		FROM branches WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&b.ID, &b.CompanyID, &b.Name, &code, &status, &b.CreatedAt, &b.UpdatedAt, &b.DeletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeBranchNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeOrgQueryFailed, err)
	}
	if b.Code, err = valueobject.NewOrgCode(code); err != nil {
		return nil, domainerr.Wrap(constant.CodeOrgQueryFailed, err)
	}
	if b.Status, err = valueobject.NewOrgStatus(status); err != nil {
		return nil, domainerr.Wrap(constant.CodeOrgQueryFailed, err)
	}
	return &b, nil
}
