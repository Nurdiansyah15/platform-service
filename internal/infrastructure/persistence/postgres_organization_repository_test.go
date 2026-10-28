package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/organization/constant"
	"platform-service/internal/domain/organization/entity"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func TestPostgresOrganizationRepositories(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		c, err := entity.NewCompany("Org", testhelper.UniqueCode("org"), time.Now())
		require.NoError(t, err)
		require.NoError(t, persistence.NewPostgresCompanyRepository(testDB).Save(ctx, c))

		branches := persistence.NewPostgresBranchRepository(testDB)
		tx := persistence.TxFromContext(ctx)

		b, err := entity.NewBranch(c.ID, entity.OrgUnitParams{Name: "B1", Code: "b1"}, time.Now())
		require.NoError(t, err)
		require.NoError(t, branches.Save(ctx, b))

		got, err := branches.FindByID(ctx, b.ID)
		require.NoError(t, err)
		assert.Equal(t, "B1", got.Code.Value())

		_, err = tx.ExecContext(ctx, `SAVEPOINT dup`)
		require.NoError(t, err)
		dupBranch, err := entity.NewBranch(c.ID, entity.OrgUnitParams{Name: "B1 lagi", Code: "B1"}, time.Now())
		require.NoError(t, err)
		testhelper.AssertDomainError(t, branches.Save(ctx, dupBranch), constant.CodeBranchCodeDuplicate)
		_, err = tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT dup`)
		require.NoError(t, err)

		// Soft delete: FindByID tidak menemukan branch yang sudah dihapus.
		now := time.Now()
		got.DeletedAt = &now
		require.NoError(t, branches.Save(ctx, got))
		_, err = branches.FindByID(ctx, b.ID)
		testhelper.AssertDomainError(t, err, constant.CodeBranchNotFound)

		// Code yang sama boleh dipakai lagi setelah soft delete (partial unique index).
		again, err := entity.NewBranch(c.ID, entity.OrgUnitParams{Name: "B1 baru", Code: "B1"}, time.Now())
		require.NoError(t, err)
		require.NoError(t, branches.Save(ctx, again))
	})
}
