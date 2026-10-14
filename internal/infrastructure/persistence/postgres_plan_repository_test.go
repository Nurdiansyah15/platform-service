package persistence_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	benefitconstant "platform-service/internal/domain/benefit/constant"
	benefitentity "platform-service/internal/domain/benefit/entity"
	"platform-service/internal/domain/plan/constant"
	"platform-service/internal/domain/plan/entity"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func seedSolutionTx(t *testing.T, ctx context.Context) string {
	t.Helper()
	var id string
	require.NoError(t, persistence.TxFromContext(ctx).QueryRowContext(ctx,
		`INSERT INTO solutions (code, name) VALUES ($1, 'S') RETURNING id`, testhelper.UniqueCode("s")).Scan(&id))
	return id
}

func seedBenefitTx(t *testing.T, ctx context.Context, solutionID string) *benefitentity.Benefit {
	t.Helper()
	b, err := benefitentity.NewBenefit(benefitentity.NewBenefitParams{
		SolutionID: solutionID, Code: "max_employees", Name: "Max", ValueType: "integer", AggregationStrategy: "sum",
	}, time.Now())
	require.NoError(t, err)
	require.NoError(t, persistence.NewPostgresBenefitRepository(testDB).Save(ctx, b))
	return b
}

func TestPostgresPlanRepositories(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		solA := seedSolutionTx(t, ctx)
		solB := seedSolutionTx(t, ctx)
		benefitA := seedBenefitTx(t, ctx, solA)
		benefitB := seedBenefitTx(t, ctx, solB)

		plans := persistence.NewPostgresPlanRepository(testDB)
		planBenefits := persistence.NewPostgresPlanBenefitRepository(testDB)
		query := persistence.NewPostgresPlanQuery(testDB)

		p, err := entity.NewPlan(entity.NewPlanParams{SolutionID: solA, Code: "pro", Name: "Pro", Price: "99999.99"}, time.Now())
		require.NoError(t, err)
		require.NoError(t, plans.Save(ctx, p))

		got, err := plans.FindByID(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, "99999.99", got.Price.String(), "NUMERIC round trip tanpa float")

		_, err = plans.FindByID(ctx, uuid.NewString())
		testhelper.AssertDomainError(t, err, constant.CodePlanNotFound)

		require.NoError(t, planBenefits.ReplaceAll(ctx, p.ID, []entity.PlanBenefit{
			{PlanID: p.ID, BenefitID: benefitA.ID, SolutionID: solA, Value: json.RawMessage(`200`)},
		}))
		items, err := query.ListPlanBenefits(ctx, p.ID)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.JSONEq(t, `200`, string(items[0].Value))

		// Composite FK menolak benefit dari solution lain walau validasi usecase terlewati.
		err = planBenefits.ReplaceAll(ctx, p.ID, []entity.PlanBenefit{
			{PlanID: p.ID, BenefitID: benefitB.ID, SolutionID: solA, Value: json.RawMessage(`1`)},
		})
		testhelper.AssertDomainError(t, err, benefitconstant.CodeBenefitSolutionMismatch)
	})
}
