package persistence_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	benefitentity "platform-service/internal/domain/benefit/entity"
	tenantentity "platform-service/internal/domain/tenant/entity"
	"platform-service/internal/domain/usage/constant"
	"platform-service/internal/domain/usage/entity"
	"platform-service/internal/domain/usage/service"
	"platform-service/internal/domain/usage/valueobject"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func TestPostgresUsageRepositories(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		tx := persistence.TxFromContext(ctx)
		solutionID := seedSolutionTx(t, ctx)
		gauge, err := benefitentity.NewBenefit(benefitentity.NewBenefitParams{
			SolutionID: solutionID, Code: "employees", Name: "E", ValueType: "integer", AggregationStrategy: "max", Metering: "gauge",
		}, time.Now())
		require.NoError(t, err)
		require.NoError(t, persistence.NewPostgresBenefitRepository(testDB).Save(ctx, gauge))

		tn, err := tenantentity.NewTenant("U", testhelper.UniqueCode("u"), time.Now())
		require.NoError(t, err)
		require.NoError(t, persistence.NewPostgresTenantRepository(testDB).Save(ctx, tn))
		var clientID string
		require.NoError(t, tx.QueryRowContext(ctx, `INSERT INTO solution_clients (solution_id, name, client_id, client_secret_hash)
			VALUES ($1, 'c', $2, 'h') RETURNING id`, solutionID, "cli_"+uuid.NewString()[:8]).Scan(&clientID))

		records := persistence.NewPostgresUsageRecordRepository(testDB)
		usage := persistence.NewPostgresTenantUsageRepository(testDB)

		report := func(qty string, at time.Time, key string) *entity.UsageRecord {
			q, err := valueobject.ParseQuantityJSON(json.RawMessage(qty))
			require.NoError(t, err)
			k, err := valueobject.NewIdempotencyKey(key)
			require.NoError(t, err)
			r, err := entity.NewUsageRecord(entity.NewUsageRecordParams{
				TenantID: tn.ID, Benefit: gauge, Mode: valueobject.ModeAbsolute, Quantity: q, IdempotencyKey: k,
				ReportedByClientID: clientID, OccurredAt: at,
			}, time.Now())
			require.NoError(t, err)
			return r
		}
		period := service.ResolvePeriod(gauge.UsageReset, time.Now(), nil, nil)
		t0 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

		r1 := report(`50`, t0, "k1")
		require.NoError(t, records.Insert(ctx, r1))
		tu, err := usage.SetAbsolute(ctx, r1, period)
		require.NoError(t, err)
		assert.Equal(t, "50.0000", tu.Used)

		stale := report(`10`, t0.Add(-time.Minute), "k2")
		tu, err = usage.SetAbsolute(ctx, stale, period)
		require.NoError(t, err)
		assert.Equal(t, "50.0000", tu.Used, "laporan basi diabaikan")

		got, err := records.FindByIdempotencyKey(ctx, solutionID, "k1")
		require.NoError(t, err)
		assert.True(t, got.SamePayload(r1))

		_, err = records.FindByIdempotencyKey(ctx, solutionID, "tidak-ada")
		testhelper.AssertDomainError(t, err, constant.CodeUsageRecordNotFound)

		testhelper.AssertDomainError(t, records.Insert(ctx, report(`1`, t0, "k1")), constant.CodeUsageDuplicateKey)
	})
}
