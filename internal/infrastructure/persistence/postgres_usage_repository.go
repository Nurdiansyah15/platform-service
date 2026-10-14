package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/usage/constant"
	"platform-service/internal/domain/usage/entity"
	"platform-service/internal/domain/usage/valueobject"
)

// PostgresUsageRecordRepository implements repository.UsageRecordRepository.
type PostgresUsageRecordRepository struct {
	db *sql.DB
}

func NewPostgresUsageRecordRepository(db *sql.DB) *PostgresUsageRecordRepository {
	return &PostgresUsageRecordRepository{db: db}
}

func (r *PostgresUsageRecordRepository) Insert(ctx context.Context, u *entity.UsageRecord) error {
	_, err := execFromContext(ctx, r.db).ExecContext(ctx, `
		INSERT INTO usage_records (id, tenant_id, solution_id, benefit_id, mode, quantity, idempotency_key,
			reported_by_client_id, occurred_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6::numeric, $7, $8, $9, $10)`,
		u.ID, u.TenantID, u.SolutionID, u.BenefitID, string(u.Mode), u.Quantity.String(), u.IdempotencyKey.Value(),
		u.ReportedByClientID, u.OccurredAt, u.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodeUsageDuplicateKey)
		}
		return domainerr.Wrap(constant.CodeUsagePersistenceFailed, err)
	}
	return nil
}

func (r *PostgresUsageRecordRepository) FindByIdempotencyKey(ctx context.Context, solutionID, key string) (*entity.UsageRecord, error) {
	var (
		u                  entity.UsageRecord
		mode, qty, idemKey string
	)
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT id, tenant_id, solution_id, benefit_id, mode, quantity::text, idempotency_key, reported_by_client_id,
			occurred_at, created_at
		FROM usage_records WHERE solution_id = $1 AND idempotency_key = $2`, solutionID, key,
	).Scan(&u.ID, &u.TenantID, &u.SolutionID, &u.BenefitID, &mode, &qty, &idemKey, &u.ReportedByClientID, &u.OccurredAt, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeUsageRecordNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeUsageQueryFailed, err)
	}
	u.Mode = valueobject.Mode(mode)
	if u.Quantity, err = valueobject.NewQuantity(qty); err != nil {
		return nil, domainerr.Wrap(constant.CodeUsageQueryFailed, err)
	}
	if u.IdempotencyKey, err = valueobject.NewIdempotencyKey(idemKey); err != nil {
		return nil, domainerr.Wrap(constant.CodeUsageQueryFailed, err)
	}
	return &u, nil
}

// PostgresTenantUsageRepository implements repository.TenantUsageRepository.
type PostgresTenantUsageRepository struct {
	db *sql.DB
}

func NewPostgresTenantUsageRepository(db *sql.DB) *PostgresTenantUsageRepository {
	return &PostgresTenantUsageRepository{db: db}
}

const tenantUsageReturning = `RETURNING tenant_id, solution_id, benefit_id, period_start, period_end, used::text, last_occurred_at, updated_at`

func (r *PostgresTenantUsageRepository) AddDelta(ctx context.Context, u *entity.UsageRecord, p entity.Period) (*entity.TenantUsage, error) {
	// Upsert atomik: aman untuk laporan paralel tanpa lock eksplisit.
	return r.scan(execFromContext(ctx, r.db).QueryRowContext(ctx, `
		INSERT INTO tenant_usage (tenant_id, solution_id, benefit_id, period_start, period_end, used, last_occurred_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6::numeric, $7, NOW())
		ON CONFLICT (tenant_id, benefit_id, period_start) DO UPDATE SET
			used = tenant_usage.used + EXCLUDED.used,
			last_occurred_at = GREATEST(tenant_usage.last_occurred_at, EXCLUDED.last_occurred_at),
			updated_at = NOW()
		`+tenantUsageReturning,
		u.TenantID, u.SolutionID, u.BenefitID, p.Start, p.End, u.Quantity.String(), u.OccurredAt))
}

func (r *PostgresTenantUsageRepository) SetAbsolute(ctx context.Context, u *entity.UsageRecord, p entity.Period) (*entity.TenantUsage, error) {
	exec := execFromContext(ctx, r.db)
	// Laporan gauge yang lebih lama dari yang sudah diterapkan diabaikan (urutan tidak dijamin).
	tu, err := r.scan(exec.QueryRowContext(ctx, `
		INSERT INTO tenant_usage (tenant_id, solution_id, benefit_id, period_start, period_end, used, last_occurred_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6::numeric, $7, NOW())
		ON CONFLICT (tenant_id, benefit_id, period_start) DO UPDATE SET
			used = EXCLUDED.used,
			last_occurred_at = EXCLUDED.last_occurred_at,
			updated_at = NOW()
		WHERE tenant_usage.last_occurred_at IS NULL OR tenant_usage.last_occurred_at <= EXCLUDED.last_occurred_at
		`+tenantUsageReturning,
		u.TenantID, u.SolutionID, u.BenefitID, p.Start, p.End, u.Quantity.String(), u.OccurredAt))
	if err == nil {
		return tu, nil
	}
	if !errors.Is(errors.Unwrap(err), sql.ErrNoRows) {
		return nil, err
	}
	// WHERE menolak update (laporan basi) → kembalikan nilai terkini.
	return r.Get(ctx, u.TenantID, u.BenefitID, p.Start)
}

// Get mengembalikan (nil, nil) bila belum ada pemakaian pada periode tersebut.
func (r *PostgresTenantUsageRepository) Get(ctx context.Context, tenantID, benefitID string, periodStart time.Time) (*entity.TenantUsage, error) {
	tu, err := r.scan(execFromContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT tenant_id, solution_id, benefit_id, period_start, period_end, used::text, last_occurred_at, updated_at
		FROM tenant_usage WHERE tenant_id = $1 AND benefit_id = $2 AND period_start = $3`, tenantID, benefitID, periodStart))
	if err != nil && errors.Is(errors.Unwrap(err), sql.ErrNoRows) {
		return nil, nil
	}
	return tu, err
}

func (r *PostgresTenantUsageRepository) scan(row *sql.Row) (*entity.TenantUsage, error) {
	var tu entity.TenantUsage
	err := row.Scan(&tu.TenantID, &tu.SolutionID, &tu.BenefitID, &tu.Period.Start, &tu.Period.End, &tu.Used, &tu.LastOccurredAt, &tu.UpdatedAt)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeUsageQueryFailed, err)
	}
	return &tu, nil
}
