package repository

import (
	"time"

	"context"

	"platform-service/internal/domain/usage/entity"
)

type UsageRecordRepository interface {
	// Insert menyimpan laporan. Key idempotensi sudah dipakai → CodeUsageDuplicateKey.
	Insert(ctx context.Context, r *entity.UsageRecord) error
	FindByIdempotencyKey(ctx context.Context, solutionID, key string) (*entity.UsageRecord, error)
}

type TenantUsageRepository interface {
	// AddDelta menambah used secara atomik (upsert) dan mengembalikan hasilnya.
	AddDelta(ctx context.Context, r *entity.UsageRecord, period entity.Period) (*entity.TenantUsage, error)
	// SetAbsolute menimpa used, kecuali laporan lebih lama dari last_occurred_at.
	SetAbsolute(ctx context.Context, r *entity.UsageRecord, period entity.Period) (*entity.TenantUsage, error)
	Get(ctx context.Context, tenantID, benefitID string, periodStart time.Time) (*entity.TenantUsage, error)
}
