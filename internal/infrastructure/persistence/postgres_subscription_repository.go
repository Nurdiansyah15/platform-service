package persistence

import (
	"context"
	"database/sql"
	"errors"

	domainerr "platform-service/internal/domain/errors"
	pricingvo "platform-service/internal/domain/pricing/valueobject"
	"platform-service/internal/domain/subscription/constant"
	"platform-service/internal/domain/subscription/entity"
	"platform-service/internal/domain/subscription/service"
	"platform-service/internal/domain/subscription/valueobject"
)

const subscriptionLiveIndex = "uq_subscriptions_live_company_solution"

// PostgresSubscriptionRepository implements repository.SubscriptionRepository.
type PostgresSubscriptionRepository struct {
	db *sql.DB
}

func NewPostgresSubscriptionRepository(db *sql.DB) *PostgresSubscriptionRepository {
	return &PostgresSubscriptionRepository{db: db}
}

func (r *PostgresSubscriptionRepository) Save(ctx context.Context, s *entity.Subscription) error {
	const q = `
		INSERT INTO subscriptions (id, company_id, solution_id, plan_id, status, billing_cycle,
			current_period_start, current_period_end, price_at_purchase, currency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::numeric, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			plan_id = EXCLUDED.plan_id,
			status = EXCLUDED.status,
			billing_cycle = EXCLUDED.billing_cycle,
			current_period_start = EXCLUDED.current_period_start,
			current_period_end = EXCLUDED.current_period_end,
			price_at_purchase = EXCLUDED.price_at_purchase,
			currency = EXCLUDED.currency
		RETURNING created_at, updated_at`

	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		s.ID, s.CompanyID, s.SolutionID, s.PlanID, s.Status.String(), s.BillingCycle.String(),
		s.CurrentPeriodStart, s.CurrentPeriodEnd, s.PriceAtPurchase.String(), s.Currency.String(), s.CreatedAt, s.UpdatedAt,
	).Scan(&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) && constraintName(err) == subscriptionLiveIndex {
			return domainerr.New(constant.CodeSubscriptionAlreadyLive)
		}
		if isForeignKeyViolation(err) {
			return domainerr.New(constant.CodeSubscriptionPlanUnavailable)
		}
		return domainerr.Wrap(constant.CodeSubscriptionPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresSubscriptionRepository) FindByID(ctx context.Context, id string) (*entity.Subscription, error) {
	return r.findOne(ctx, `WHERE id = $1`, id)
}

func (r *PostgresSubscriptionRepository) FindUsable(ctx context.Context, companyID, solutionID string) (*entity.Subscription, error) {
	return r.findOne(ctx, `WHERE company_id = $1 AND solution_id = $2 AND status = ANY($3::text[])`,
		companyID, solutionID, valueobject.UsableStatuses)
}

func (r *PostgresSubscriptionRepository) findOne(ctx context.Context, where string, args ...any) (*entity.Subscription, error) {
	var (
		s                              entity.Subscription
		status, cycle, price, currency string
	)
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT id, company_id, solution_id, plan_id, status, billing_cycle, current_period_start, current_period_end,
			price_at_purchase::text, currency, created_at, updated_at
		FROM subscriptions `+where, args...,
	).Scan(&s.ID, &s.CompanyID, &s.SolutionID, &s.PlanID, &status, &cycle, &s.CurrentPeriodStart, &s.CurrentPeriodEnd,
		&price, &currency, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeSubscriptionNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	if s.Status, err = valueobject.NewSubscriptionStatus(status); err != nil {
		return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	if s.BillingCycle, err = pricingvo.NewBillingCycle(cycle); err != nil {
		return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	if s.PriceAtPurchase, err = pricingvo.NewPrice(price); err != nil {
		return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	if s.Currency, err = pricingvo.NewCurrency(currency); err != nil {
		return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	return &s, nil
}

func (r *PostgresSubscriptionRepository) HasUsable(ctx context.Context, companyID, solutionID string) (bool, error) {
	var ok bool
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM subscriptions
			WHERE company_id = $1 AND solution_id = $2 AND status = ANY($3::text[]))`,
		companyID, solutionID, valueobject.UsableStatuses,
	).Scan(&ok)
	if err != nil {
		return false, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	return ok, nil
}

// PostgresSubscriptionAddonRepository implements repository.SubscriptionAddonRepository.
type PostgresSubscriptionAddonRepository struct {
	db *sql.DB
}

func NewPostgresSubscriptionAddonRepository(db *sql.DB) *PostgresSubscriptionAddonRepository {
	return &PostgresSubscriptionAddonRepository{db: db}
}

func (r *PostgresSubscriptionAddonRepository) Save(ctx context.Context, a *entity.SubscriptionAddon) error {
	const q = `
		INSERT INTO subscription_addons (id, subscription_id, addon_id, solution_id, quantity, unit_price_at_purchase,
			currency, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6::numeric, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			quantity = EXCLUDED.quantity,
			unit_price_at_purchase = EXCLUDED.unit_price_at_purchase,
			currency = EXCLUDED.currency,
			status = EXCLUDED.status
		RETURNING created_at, updated_at`

	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		a.ID, a.SubscriptionID, a.AddonID, a.SolutionID, a.Quantity, a.UnitPriceAtPurchase.String(),
		a.Currency.String(), string(a.Status), a.CreatedAt, a.UpdatedAt,
	).Scan(&a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodeSubscriptionAddonAlreadyActive)
		}
		if isForeignKeyViolation(err) {
			return domainerr.New(constant.CodeSubscriptionAddonUnavailable)
		}
		return domainerr.Wrap(constant.CodeSubscriptionPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresSubscriptionAddonRepository) Find(ctx context.Context, subscriptionID, addonID string) (*entity.SubscriptionAddon, error) {
	var (
		a                       entity.SubscriptionAddon
		price, currency, status string
	)
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT id, subscription_id, addon_id, solution_id, quantity, unit_price_at_purchase::text, currency, status,
			created_at, updated_at
		FROM subscription_addons WHERE subscription_id = $1 AND addon_id = $2`, subscriptionID, addonID,
	).Scan(&a.ID, &a.SubscriptionID, &a.AddonID, &a.SolutionID, &a.Quantity, &price, &currency, &status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeSubscriptionAddonNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	if a.UnitPriceAtPurchase, err = pricingvo.NewPrice(price); err != nil {
		return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	if a.Currency, err = pricingvo.NewCurrency(currency); err != nil {
		return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	a.Status = entity.SubscriptionAddonStatus(status)
	return &a, nil
}

// PostgresSnapshotRepository implements repository.SnapshotRepository.
type PostgresSnapshotRepository struct {
	db *sql.DB
}

func NewPostgresSnapshotRepository(db *sql.DB) *PostgresSnapshotRepository {
	return &PostgresSnapshotRepository{db: db}
}

func (r *PostgresSnapshotRepository) ReplacePlanSnapshots(ctx context.Context, subscriptionID string, rows []entity.BenefitSnapshot) error {
	return r.replace(ctx, `DELETE FROM subscription_benefit_snapshots WHERE subscription_id = $1 AND subscription_addon_id IS NULL`, subscriptionID, rows)
}

func (r *PostgresSnapshotRepository) ReplaceAddonSnapshots(ctx context.Context, subscriptionAddonID string, rows []entity.BenefitSnapshot) error {
	return r.replace(ctx, `DELETE FROM subscription_benefit_snapshots WHERE subscription_addon_id = $1`, subscriptionAddonID, rows)
}

func (r *PostgresSnapshotRepository) replace(ctx context.Context, deleteQ, key string, rows []entity.BenefitSnapshot) error {
	exec := execFromContext(ctx, r.db)
	if _, err := exec.ExecContext(ctx, deleteQ, key); err != nil {
		return domainerr.Wrap(constant.CodeSubscriptionPersistenceFailed, err)
	}
	for _, s := range rows {
		_, err := exec.ExecContext(ctx, `
			INSERT INTO subscription_benefit_snapshots (id, subscription_id, subscription_addon_id, benefit_id, value, created_at)
			VALUES ($1, $2, $3, $4, $5::jsonb, $6)`,
			s.ID, s.SubscriptionID, s.SubscriptionAddonID, s.BenefitID, string(s.Value), s.CreatedAt)
		if err != nil {
			return domainerr.Wrap(constant.CodeSubscriptionPersistenceFailed, err)
		}
	}
	return nil
}

// PostgresBenefitValueSource implements service.BenefitValueSource.
type PostgresBenefitValueSource struct {
	db *sql.DB
}

func NewPostgresBenefitValueSource(db *sql.DB) *PostgresBenefitValueSource {
	return &PostgresBenefitValueSource{db: db}
}

func (r *PostgresBenefitValueSource) PlanBenefitValues(ctx context.Context, planID string) ([]service.BenefitValue, error) {
	return r.values(ctx, `SELECT benefit_id, value FROM plan_benefits WHERE plan_id = $1 ORDER BY benefit_id`, planID)
}

func (r *PostgresBenefitValueSource) AddonBenefitValues(ctx context.Context, addonID string) ([]service.BenefitValue, error) {
	return r.values(ctx, `SELECT benefit_id, value FROM addon_benefits WHERE addon_id = $1 ORDER BY benefit_id`, addonID)
}

func (r *PostgresBenefitValueSource) values(ctx context.Context, q, id string) ([]service.BenefitValue, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx, q, id)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	defer rows.Close()
	var out []service.BenefitValue
	for rows.Next() {
		var (
			v     service.BenefitValue
			value []byte
		)
		if err := rows.Scan(&v.BenefitID, &value); err != nil {
			return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
		}
		v.Value = value
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeSubscriptionQueryFailed, err)
	}
	return out, nil
}
