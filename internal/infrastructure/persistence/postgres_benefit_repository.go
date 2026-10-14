package persistence

import (
	"context"
	"database/sql"
	"errors"

	"platform-service/internal/domain/benefit/constant"
	"platform-service/internal/domain/benefit/entity"
	"platform-service/internal/domain/benefit/valueobject"
	domainerr "platform-service/internal/domain/errors"
)

// PostgresBenefitRepository implements repository.BenefitRepository.
type PostgresBenefitRepository struct {
	db *sql.DB
}

func NewPostgresBenefitRepository(db *sql.DB) *PostgresBenefitRepository {
	return &PostgresBenefitRepository{db: db}
}

const benefitColumns = `id, solution_id, code, name, description, value_type, aggregation_strategy,
	unit, metering, usage_reset, created_at, updated_at`

func (r *PostgresBenefitRepository) Save(ctx context.Context, b *entity.Benefit) error {
	const q = `
		INSERT INTO benefits (id, solution_id, code, name, description, value_type, aggregation_strategy,
			unit, metering, usage_reset, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			aggregation_strategy = EXCLUDED.aggregation_strategy,
			unit = EXCLUDED.unit,
			metering = EXCLUDED.metering,
			usage_reset = EXCLUDED.usage_reset
		RETURNING created_at, updated_at`

	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		b.ID, b.SolutionID, b.Code.Value(), b.Name, b.Description, string(b.ValueType), string(b.AggregationStrategy),
		b.Unit, string(b.Metering), string(b.UsageReset), b.CreatedAt, b.UpdatedAt,
	).Scan(&b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodeBenefitCodeDuplicate)
		}
		return domainerr.Wrap(constant.CodeBenefitPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresBenefitRepository) FindByID(ctx context.Context, id string) (*entity.Benefit, error) {
	b, err := scanBenefit(execFromContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT `+benefitColumns+` FROM benefits WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeBenefitNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeBenefitQueryFailed, err)
	}
	return b, nil
}

func (r *PostgresBenefitRepository) FindByCode(ctx context.Context, solutionID, code string) (*entity.Benefit, error) {
	b, err := scanBenefit(execFromContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT `+benefitColumns+` FROM benefits WHERE solution_id = $1 AND code = $2`, solutionID, code))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeBenefitNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeBenefitQueryFailed, err)
	}
	return b, nil
}

func (r *PostgresBenefitRepository) FindByIDs(ctx context.Context, ids []string) ([]*entity.Benefit, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx,
		`SELECT `+benefitColumns+` FROM benefits WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeBenefitQueryFailed, err)
	}
	defer rows.Close()

	out := make([]*entity.Benefit, 0, len(ids))
	for rows.Next() {
		b, err := scanBenefit(rows)
		if err != nil {
			return nil, domainerr.Wrap(constant.CodeBenefitQueryFailed, err)
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeBenefitQueryFailed, err)
	}
	return out, nil
}

func scanBenefit(row interface{ Scan(dest ...any) error }) (*entity.Benefit, error) {
	var (
		b                                             entity.Benefit
		code, valueType, aggregation, metering, reset string
	)
	if err := row.Scan(&b.ID, &b.SolutionID, &code, &b.Name, &b.Description, &valueType, &aggregation,
		&b.Unit, &metering, &reset, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, err
	}
	var err error
	if b.Code, err = valueobject.NewBenefitCode(code); err != nil {
		return nil, err
	}
	if b.ValueType, err = valueobject.NewValueType(valueType); err != nil {
		return nil, err
	}
	if b.AggregationStrategy, err = valueobject.NewAggregationStrategy(aggregation); err != nil {
		return nil, err
	}
	if b.Metering, err = valueobject.NewMetering(metering); err != nil {
		return nil, err
	}
	if b.UsageReset, err = valueobject.NewUsageReset(reset); err != nil {
		return nil, err
	}
	return &b, nil
}
