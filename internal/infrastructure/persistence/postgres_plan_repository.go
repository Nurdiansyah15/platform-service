package persistence

import (
	"context"
	"database/sql"
	"errors"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/plan/constant"
	"platform-service/internal/domain/plan/entity"
	planvo "platform-service/internal/domain/plan/valueobject"
	pricingvo "platform-service/internal/domain/pricing/valueobject"
)

// PostgresPlanRepository implements repository.PlanRepository.
type PostgresPlanRepository struct {
	db *sql.DB
}

func NewPostgresPlanRepository(db *sql.DB) *PostgresPlanRepository {
	return &PostgresPlanRepository{db: db}
}

// price di-cast ke TEXT agar presisi NUMERIC terjaga (tanpa float).
const planColumns = `id, solution_id, code, name, description, price::text, currency, billing_cycle,
	is_active, created_at, updated_at, deleted_at`

func (r *PostgresPlanRepository) Save(ctx context.Context, p *entity.Plan) error {
	const q = `
		INSERT INTO plans (id, solution_id, code, name, description, price, currency, billing_cycle,
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
		p.ID, p.SolutionID, p.Code.Value(), p.Name, p.Description, p.Price.String(), p.Currency.String(),
		p.BillingCycle.String(), p.IsActive, p.CreatedAt, p.UpdatedAt, p.DeletedAt,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodePlanCodeDuplicate)
		}
		return domainerr.Wrap(constant.CodePlanPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresPlanRepository) FindByID(ctx context.Context, id string) (*entity.Plan, error) {
	var (
		p                            entity.Plan
		code, price, currency, cycle string
	)
	err := execFromContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT `+planColumns+` FROM plans WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&p.ID, &p.SolutionID, &code, &p.Name, &p.Description, &price, &currency, &cycle,
		&p.IsActive, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodePlanNotFound)
		}
		return nil, domainerr.Wrap(constant.CodePlanQueryFailed, err)
	}
	if p.Code, err = planvo.NewPlanCode(code); err != nil {
		return nil, domainerr.Wrap(constant.CodePlanQueryFailed, err)
	}
	if p.Price, err = pricingvo.NewPrice(price); err != nil {
		return nil, domainerr.Wrap(constant.CodePlanQueryFailed, err)
	}
	if p.Currency, err = pricingvo.NewCurrency(currency); err != nil {
		return nil, domainerr.Wrap(constant.CodePlanQueryFailed, err)
	}
	if p.BillingCycle, err = pricingvo.NewBillingCycle(cycle); err != nil {
		return nil, domainerr.Wrap(constant.CodePlanQueryFailed, err)
	}
	return &p, nil
}

// PostgresPlanBenefitRepository implements repository.PlanBenefitRepository.
type PostgresPlanBenefitRepository struct {
	db *sql.DB
}

func NewPostgresPlanBenefitRepository(db *sql.DB) *PostgresPlanBenefitRepository {
	return &PostgresPlanBenefitRepository{db: db}
}

func (r *PostgresPlanBenefitRepository) ReplaceAll(ctx context.Context, planID string, items []entity.PlanBenefit) error {
	exec := execFromContext(ctx, r.db)
	if _, err := exec.ExecContext(ctx, `DELETE FROM plan_benefits WHERE plan_id = $1`, planID); err != nil {
		return domainerr.Wrap(constant.CodePlanPersistenceFailed, err)
	}
	for _, it := range items {
		_, err := exec.ExecContext(ctx,
			`INSERT INTO plan_benefits (plan_id, benefit_id, solution_id, value) VALUES ($1, $2, $3, $4::jsonb)`,
			it.PlanID, it.BenefitID, it.SolutionID, string(it.Value))
		if err != nil {
			if isForeignKeyViolation(err) {
				return domainerr.New(benefitSolutionMismatch)
			}
			return domainerr.Wrap(constant.CodePlanPersistenceFailed, err)
		}
	}
	return nil
}
