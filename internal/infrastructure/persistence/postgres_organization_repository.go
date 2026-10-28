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

// PostgresCompanyRepository implements repository.CompanyRepository (company = entitas tertinggi).
type PostgresCompanyRepository struct {
	db *sql.DB
}

func NewPostgresCompanyRepository(db *sql.DB) *PostgresCompanyRepository {
	return &PostgresCompanyRepository{db: db}
}

func (r *PostgresCompanyRepository) Save(ctx context.Context, c *entity.Company) error {
	const q = `
		INSERT INTO companies (id, name, slug, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			slug = EXCLUDED.slug,
			status = EXCLUDED.status
		RETURNING created_at, updated_at`

	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		c.ID, c.Name, c.Slug.Value(), c.Status.String(), c.CreatedAt, c.UpdatedAt,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodeCompanySlugDuplicate)
		}
		return domainerr.Wrap(constant.CodeCompanyPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresCompanyRepository) FindByID(ctx context.Context, id string) (*entity.Company, error) {
	var (
		c            entity.Company
		slug, status string
	)
	err := execFromContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT id, name, slug, status, created_at, updated_at FROM companies WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &slug, &status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeCompanyNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
	}
	if c.Slug, err = valueobject.NewSlug(slug); err != nil {
		return nil, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
	}
	if c.Status, err = valueobject.NewCompanyStatus(status); err != nil {
		return nil, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
	}
	return &c, nil
}

// PostgresCompanyMemberRepository implements repository.CompanyMemberRepository.
type PostgresCompanyMemberRepository struct {
	db *sql.DB
}

func NewPostgresCompanyMemberRepository(db *sql.DB) *PostgresCompanyMemberRepository {
	return &PostgresCompanyMemberRepository{db: db}
}

func (r *PostgresCompanyMemberRepository) Save(ctx context.Context, m *entity.CompanyMember) error {
	const q = `
		INSERT INTO company_members (company_id, user_id, status, joined_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (company_id, user_id) DO UPDATE SET
			status = EXCLUDED.status,
			joined_at = EXCLUDED.joined_at`

	if _, err := execFromContext(ctx, r.db).ExecContext(ctx, q, m.CompanyID, m.UserID, m.Status.String(), m.JoinedAt); err != nil {
		return domainerr.Wrap(constant.CodeCompanyPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresCompanyMemberRepository) Find(ctx context.Context, companyID, userID string) (*entity.CompanyMember, error) {
	var (
		m      entity.CompanyMember
		status string
	)
	err := execFromContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT company_id, user_id, status, joined_at FROM company_members WHERE company_id = $1 AND user_id = $2`,
		companyID, userID,
	).Scan(&m.CompanyID, &m.UserID, &status, &m.JoinedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeMemberNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
	}
	if m.Status, err = valueobject.NewMemberStatus(status); err != nil {
		return nil, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
	}
	return &m, nil
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
