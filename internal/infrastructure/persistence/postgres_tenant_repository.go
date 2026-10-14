package persistence

import (
	"context"
	"database/sql"
	"errors"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/tenant/constant"
	"platform-service/internal/domain/tenant/entity"
	"platform-service/internal/domain/tenant/valueobject"
)

// PostgresTenantRepository implements repository.TenantRepository.
type PostgresTenantRepository struct {
	db *sql.DB
}

func NewPostgresTenantRepository(db *sql.DB) *PostgresTenantRepository {
	return &PostgresTenantRepository{db: db}
}

func (r *PostgresTenantRepository) Save(ctx context.Context, t *entity.Tenant) error {
	const q = `
		INSERT INTO tenants (id, name, slug, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			slug = EXCLUDED.slug,
			status = EXCLUDED.status
		RETURNING created_at, updated_at`

	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		t.ID, t.Name, t.Slug.Value(), t.Status.String(), t.CreatedAt, t.UpdatedAt,
	).Scan(&t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodeTenantSlugDuplicate)
		}
		return domainerr.Wrap(constant.CodeTenantPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresTenantRepository) FindByID(ctx context.Context, id string) (*entity.Tenant, error) {
	var (
		t            entity.Tenant
		slug, status string
	)
	err := execFromContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT id, name, slug, status, created_at, updated_at FROM tenants WHERE id = $1`, id,
	).Scan(&t.ID, &t.Name, &slug, &status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeTenantNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeTenantQueryFailed, err)
	}
	if t.Slug, err = valueobject.NewSlug(slug); err != nil {
		return nil, domainerr.Wrap(constant.CodeTenantQueryFailed, err)
	}
	if t.Status, err = valueobject.NewTenantStatus(status); err != nil {
		return nil, domainerr.Wrap(constant.CodeTenantQueryFailed, err)
	}
	return &t, nil
}

// PostgresTenantMemberRepository implements repository.TenantMemberRepository.
type PostgresTenantMemberRepository struct {
	db *sql.DB
}

func NewPostgresTenantMemberRepository(db *sql.DB) *PostgresTenantMemberRepository {
	return &PostgresTenantMemberRepository{db: db}
}

func (r *PostgresTenantMemberRepository) Save(ctx context.Context, m *entity.TenantMember) error {
	const q = `
		INSERT INTO tenant_members (tenant_id, user_id, status, joined_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tenant_id, user_id) DO UPDATE SET
			status = EXCLUDED.status,
			joined_at = EXCLUDED.joined_at`

	if _, err := execFromContext(ctx, r.db).ExecContext(ctx, q, m.TenantID, m.UserID, m.Status.String(), m.JoinedAt); err != nil {
		return domainerr.Wrap(constant.CodeTenantPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresTenantMemberRepository) Find(ctx context.Context, tenantID, userID string) (*entity.TenantMember, error) {
	var (
		m      entity.TenantMember
		status string
	)
	err := execFromContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT tenant_id, user_id, status, joined_at FROM tenant_members WHERE tenant_id = $1 AND user_id = $2`,
		tenantID, userID,
	).Scan(&m.TenantID, &m.UserID, &status, &m.JoinedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeMemberNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeTenantQueryFailed, err)
	}
	if m.Status, err = valueobject.NewMemberStatus(status); err != nil {
		return nil, domainerr.Wrap(constant.CodeTenantQueryFailed, err)
	}
	return &m, nil
}
