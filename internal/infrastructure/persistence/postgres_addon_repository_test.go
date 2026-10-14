package persistence_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/addon/constant"
	"platform-service/internal/domain/addon/entity"
	benefitconstant "platform-service/internal/domain/benefit/constant"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func TestPostgresAddonRepositories(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		solA := seedSolutionTx(t, ctx)
		solB := seedSolutionTx(t, ctx)
		benefitA := seedBenefitTx(t, ctx, solA)
		benefitB := seedBenefitTx(t, ctx, solB)

		addons := persistence.NewPostgresAddonRepository(testDB)
		addonBenefits := persistence.NewPostgresAddonBenefitRepository(testDB)

		a, err := entity.NewAddon(entity.NewAddonParams{SolutionID: solA, Code: "payroll", Name: "Payroll", Price: "25000.5"}, time.Now())
		require.NoError(t, err)
		require.NoError(t, addons.Save(ctx, a))

		got, err := addons.FindByID(ctx, a.ID)
		require.NoError(t, err)
		assert.Equal(t, "25000.50", got.Price.String())

		_, err = addons.FindByID(ctx, uuid.NewString())
		testhelper.AssertDomainError(t, err, constant.CodeAddonNotFound)

		require.NoError(t, addonBenefits.ReplaceAll(ctx, a.ID, []entity.AddonBenefit{
			{AddonID: a.ID, BenefitID: benefitA.ID, SolutionID: solA, Value: json.RawMessage(`10`)},
		}))
		items, err := persistence.NewPostgresAddonQuery(testDB).ListAddonBenefits(ctx, a.ID)
		require.NoError(t, err)
		require.Len(t, items, 1)

		err = addonBenefits.ReplaceAll(ctx, a.ID, []entity.AddonBenefit{
			{AddonID: a.ID, BenefitID: benefitB.ID, SolutionID: solA, Value: json.RawMessage(`1`)},
		})
		testhelper.AssertDomainError(t, err, benefitconstant.CodeBenefitSolutionMismatch)
	})
}
