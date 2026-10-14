package persistence

import (
	"context"
	"database/sql"
	"errors"

	"platform-service/internal/domain/addon/constant"
	"platform-service/internal/domain/addon/entity"
	addonvo "platform-service/internal/domain/addon/valueobject"
	domainerr "platform-service/internal/domain/errors"
	pricingvo "platform-service/internal/domain/pricing/valueobject"
)

// PostgresAddonRepository implements repository.AddonRepository.
type PostgresAddonRepository struct {
	db *sql.DB
}

func NewPostgresAddonRepository(db *sql.DB) *PostgresAddonRepository {
	return &PostgresAddonRepository{db: db}
}

func (r *PostgresAddonRepository) Save(ctx context.Context, a *entity.Addon) error {
	const q = `
		INSERT INTO addons (id, solution_id, code, name, description, price, currency, billing_cycle,
			is_active, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6::numeric, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			code = EXCLUDED.code,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			price = EXCLUDED.price,
			currency = EXCLUDED.currency,
			billing_cycle = EXCLUDED.billing_cycle,
			is_active = EXCLUDED.is_active,
			deleted_at = EXCLUDED.deleted_at
		RETURNING created_at, updated_at`

	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		a.ID, a.SolutionID, a.Code.Value(), a.Name, a.Description, a.Price.String(), a.Currency.String(),
		a.BillingCycle.String(), a.IsActive, a.CreatedAt, a.UpdatedAt, a.DeletedAt,
	).Scan(&a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodeAddonCodeDuplicate)
		}
		return domainerr.Wrap(constant.CodeAddonPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresAddonRepository) FindByID(ctx context.Context, id string) (*entity.Addon, error) {
	var (
		a                            entity.Addon
		code, price, currency, cycle string
	)
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT id, solution_id, code, name, description, price::text, currency, billing_cycle,
			is_active, created_at, updated_at, deleted_at
		FROM addons WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&a.ID, &a.SolutionID, &code, &a.Name, &a.Description, &price, &currency, &cycle,
		&a.IsActive, &a.CreatedAt, &a.UpdatedAt, &a.DeletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeAddonNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeAddonQueryFailed, err)
	}
	if a.Code, err = addonvo.NewAddonCode(code); err != nil {
		return nil, domainerr.Wrap(constant.CodeAddonQueryFailed, err)
	}
	if a.Price, err = pricingvo.NewPrice(price); err != nil {
		return nil, domainerr.Wrap(constant.CodeAddonQueryFailed, err)
	}
	if a.Currency, err = pricingvo.NewCurrency(currency); err != nil {
		return nil, domainerr.Wrap(constant.CodeAddonQueryFailed, err)
	}
	if a.BillingCycle, err = pricingvo.NewBillingCycle(cycle); err != nil {
		return nil, domainerr.Wrap(constant.CodeAddonQueryFailed, err)
	}
	return &a, nil
}

// PostgresAddonBenefitRepository implements repository.AddonBenefitRepository.
type PostgresAddonBenefitRepository struct {
	db *sql.DB
}

func NewPostgresAddonBenefitRepository(db *sql.DB) *PostgresAddonBenefitRepository {
	return &PostgresAddonBenefitRepository{db: db}
}

func (r *PostgresAddonBenefitRepository) ReplaceAll(ctx context.Context, addonID string, items []entity.AddonBenefit) error {
	exec := execFromContext(ctx, r.db)
	if _, err := exec.ExecContext(ctx, `DELETE FROM addon_benefits WHERE addon_id = $1`, addonID); err != nil {
		return domainerr.Wrap(constant.CodeAddonPersistenceFailed, err)
	}
	for _, it := range items {
		_, err := exec.ExecContext(ctx,
			`INSERT INTO addon_benefits (addon_id, benefit_id, solution_id, value) VALUES ($1, $2, $3, $4::jsonb)`,
			it.AddonID, it.BenefitID, it.SolutionID, string(it.Value))
		if err != nil {
			if isForeignKeyViolation(err) {
				return domainerr.New(benefitSolutionMismatch)
			}
			return domainerr.Wrap(constant.CodeAddonPersistenceFailed, err)
		}
	}
	return nil
}
