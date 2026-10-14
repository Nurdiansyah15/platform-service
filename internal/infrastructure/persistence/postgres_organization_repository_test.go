package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/organization/constant"
	"platform-service/internal/domain/organization/entity"
	tenantentity "platform-service/internal/domain/tenant/entity"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func TestPostgresOrganizationRepositories(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		tn, err := tenantentity.NewTenant("Org", testhelper.UniqueCode("org"), time.Now())
		require.NoError(t, err)
		require.NoError(t, persistence.NewPostgresTenantRepository(testDB).Save(ctx, tn))

		companies := persistence.NewPostgresCompanyRepository(testDB)
		branches := persistence.NewPostgresBranchRepository(testDB)
		tx := persistence.TxFromContext(ctx)

		c, err := entity.NewCompany(tn.ID, entity.OrgUnitParams{Name: "HQ", Code: "hq"}, time.Now())
		require.NoError(t, err)
		require.NoError(t, companies.Save(ctx, c))

		got, err := companies.FindByID(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, "HQ", got.Code.Value())

		b, err := entity.NewBranch(c.ID, entity.OrgUnitParams{Name: "B1", Code: "b1"}, time.Now())
		require.NoError(t, err)
		require.NoError(t, branches.Save(ctx, b))

		_, err = tx.ExecContext(ctx, `SAVEPOINT dup`)
		require.NoError(t, err)
		dupBranch, err := entity.NewBranch(c.ID, entity.OrgUnitParams{Name: "B1 lagi", Code: "B1"}, time.Now())
		require.NoError(t, err)
		testhelper.AssertDomainError(t, branches.Save(ctx, dupBranch), constant.CodeBranchCodeDuplicate)
		_, err = tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT dup`)
		require.NoError(t, err)

		// Soft delete: FindByID tidak menemukan company yang sudah dihapus.
		now := time.Now()
		got.DeletedAt = &now
		require.NoError(t, companies.Save(ctx, got))
		_, err = companies.FindByID(ctx, c.ID)
		testhelper.AssertDomainError(t, err, constant.CodeCompanyNotFound)

		// Code yang sama boleh dipakai lagi setelah soft delete (partial unique index).
		again, err := entity.NewCompany(tn.ID, entity.OrgUnitParams{Name: "HQ baru", Code: "HQ"}, time.Now())
		require.NoError(t, err)
		require.NoError(t, companies.Save(ctx, again))

		dupCompany, err := entity.NewCompany(tn.ID, entity.OrgUnitParams{Name: "HQ dup", Code: "HQ"}, time.Now())
		require.NoError(t, err)
		testhelper.AssertDomainError(t, companies.Save(ctx, dupCompany), constant.CodeCompanyCodeDuplicate)
	})
}
