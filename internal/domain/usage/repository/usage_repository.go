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

type CompanyUsageRepository interface {
	// AddDelta menambah used secara atomik (upsert) dan mengembalikan hasilnya.
	AddDelta(ctx context.Context, r *entity.UsageRecord, period entity.Period) (*entity.CompanyUsage, error)
	// SetAbsolute menimpa used, kecuali laporan lebih lama dari last_occurred_at.
	SetAbsolute(ctx context.Context, r *entity.UsageRecord, period entity.Period) (*entity.CompanyUsage, error)
	// Get mengembalikan (nil, nil) bila belum ada pemakaian pada periode tersebut.
	Get(ctx context.Context, companyID, benefitID string, periodStart time.Time) (*entity.CompanyUsage, error)
}
