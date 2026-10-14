package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/app/port"
	"platform-service/internal/domain/benefit/constant"
	"platform-service/internal/domain/benefit/entity"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func TestPostgresBenefitRepository(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		var solutionID string
		require.NoError(t, persistence.TxFromContext(ctx).QueryRowContext(ctx,
			`INSERT INTO solutions (code, name) VALUES ($1, 'S') RETURNING id`, testhelper.UniqueCode("bn")).Scan(&solutionID))

		repo := persistence.NewPostgresBenefitRepository(testDB)
		unit := "call"
		b, err := entity.NewBenefit(entity.NewBenefitParams{
			SolutionID: solutionID, Code: "api_calls", Name: "API Calls", ValueType: "integer",
			AggregationStrategy: "sum", Unit: &unit, Metering: "counter", UsageReset: "monthly",
		}, time.Now())
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, b))

		got, err := repo.FindByID(ctx, b.ID)
		require.NoError(t, err)
		assert.Equal(t, "api_calls", got.Code.Value())
		assert.True(t, got.IsMetered())
		assert.Equal(t, "call", *got.Unit)

		items, meta, err := persistence.NewPostgresBenefitQuery(testDB).List(ctx, port.BenefitListReadQuery{SolutionID: solutionID})
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "counter", items[0].Metering)
		assert.Equal(t, int64(1), meta.Total)

		_, err = repo.FindByID(ctx, uuid.NewString())
		testhelper.AssertDomainError(t, err, constant.CodeBenefitNotFound)

		dup, err := entity.NewBenefit(entity.NewBenefitParams{
			SolutionID: solutionID, Code: "api_calls", Name: "Dup", ValueType: "integer", AggregationStrategy: "sum",
		}, time.Now())
		require.NoError(t, err)
		testhelper.AssertDomainError(t, repo.Save(ctx, dup), constant.CodeBenefitCodeDuplicate)
	})
}
