package persistence_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planentity "platform-service/internal/domain/plan/entity"
	"platform-service/internal/domain/subscription/constant"
	"platform-service/internal/domain/subscription/entity"
	tenantentity "platform-service/internal/domain/tenant/entity"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func TestPostgresSubscriptionRepositories(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		tx := persistence.TxFromContext(ctx)
		solutionID := seedSolutionTx(t, ctx)
		benefit := seedBenefitTx(t, ctx, solutionID)

		tn, err := tenantentity.NewTenant("Sub", testhelper.UniqueCode("sub"), time.Now())
		require.NoError(t, err)
		require.NoError(t, persistence.NewPostgresTenantRepository(testDB).Save(ctx, tn))

		p, err := planentity.NewPlan(planentity.NewPlanParams{SolutionID: solutionID, Code: "basic", Name: "Basic", Price: "1000"}, time.Now())
		require.NoError(t, err)
		require.NoError(t, persistence.NewPostgresPlanRepository(testDB).Save(ctx, p))

		subs := persistence.NewPostgresSubscriptionRepository(testDB)
		snapshots := persistence.NewPostgresSnapshotRepository(testDB)
		terms := entity.PlanTerms{PlanID: p.ID, SolutionID: solutionID, BillingCycle: p.BillingCycle, Price: p.Price, Currency: p.Currency}

		ok, err := subs.HasUsable(ctx, tn.ID, solutionID)
		require.NoError(t, err)
		assert.False(t, ok)

		s, err := entity.NewSubscription(tn.ID, terms, "", time.Now(), time.Now())
		require.NoError(t, err)
		require.NoError(t, subs.Save(ctx, s))

		got, err := subs.FindByID(ctx, s.ID)
		require.NoError(t, err)
		assert.Equal(t, "1000.00", got.PriceAtPurchase.String())
		ok, err = subs.HasUsable(ctx, tn.ID, solutionID)
		require.NoError(t, err)
		assert.True(t, ok)

		// Snapshot plan: replace mengganti isi; NULLS NOT DISTINCT menolak duplikat benefit plan.
		row := entity.BenefitSnapshot{ID: "00000000-0000-0000-0000-00000000000a", SubscriptionID: s.ID, BenefitID: benefit.ID, Value: json.RawMessage(`1`), CreatedAt: time.Now()}
		require.NoError(t, snapshots.ReplacePlanSnapshots(ctx, s.ID, []entity.BenefitSnapshot{row}))
		_, err = tx.ExecContext(ctx, `SAVEPOINT dup_snapshot`)
		require.NoError(t, err)
		_, err = tx.ExecContext(ctx, `INSERT INTO subscription_benefit_snapshots (subscription_id, subscription_addon_id, benefit_id, value)
			VALUES ($1, NULL, $2, '2')`, s.ID, benefit.ID)
		assert.Error(t, err, "UNIQUE NULLS NOT DISTINCT menolak snapshot plan ganda untuk benefit yang sama")
		_, err = tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT dup_snapshot`)
		require.NoError(t, err)

		// Subscription live kedua untuk tenant+solution ditolak.
		second, err := entity.NewSubscription(tn.ID, terms, "", time.Now(), time.Now())
		require.NoError(t, err)
		_, err = tx.ExecContext(ctx, `SAVEPOINT dup_live`)
		require.NoError(t, err)
		testhelper.AssertDomainError(t, subs.Save(ctx, second), constant.CodeSubscriptionAlreadyLive)
		_, err = tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT dup_live`)
		require.NoError(t, err)

		// Setelah dibatalkan, subscription baru boleh dibuat.
		require.NoError(t, got.Cancel(time.Now()))
		require.NoError(t, subs.Save(ctx, got))
		require.NoError(t, subs.Save(ctx, second))
	})
}
