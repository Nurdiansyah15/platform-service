package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	companyentity "platform-service/internal/domain/organization/entity"
	permentity "platform-service/internal/domain/permission/entity"
	"platform-service/internal/domain/role/constant"
	"platform-service/internal/domain/role/entity"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func TestPostgresRoleRepositories(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		tx := persistence.TxFromContext(ctx)
		savepoint := func(name string, fn func()) {
			_, err := tx.ExecContext(ctx, `SAVEPOINT `+name)
			require.NoError(t, err)
			fn()
			_, err = tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT `+name)
			require.NoError(t, err)
		}

		solA, solB := seedSolutionTx(t, ctx), seedSolutionTx(t, ctx)
		tn, err := companyentity.NewCompany("R", testhelper.UniqueCode("r"), time.Now())
		require.NoError(t, err)
		require.NoError(t, persistence.NewPostgresCompanyRepository(testDB).Save(ctx, tn))

		roles := persistence.NewPostgresRoleRepository(testDB)
		rolePerms := persistence.NewPostgresRolePermissionRepository(testDB)
		perms := persistence.NewPostgresPermissionRepository(testDB)

		global, err := entity.NewRole(entity.NewRoleParams{SolutionID: solA, Code: "viewer", Name: "Viewer"}, time.Now())
		require.NoError(t, err)
		require.NoError(t, roles.Save(ctx, global))
		custom, err := entity.NewRole(entity.NewRoleParams{SolutionID: solA, CompanyID: &tn.ID, Code: "viewer", Name: "Viewer"}, time.Now())
		require.NoError(t, err)
		require.NoError(t, roles.Save(ctx, custom), "code sama boleh: global vs custom company")

		savepoint("dup_global", func() {
			dup, err := entity.NewRole(entity.NewRoleParams{SolutionID: solA, Code: "viewer", Name: "Dup"}, time.Now())
			require.NoError(t, err)
			testhelper.AssertDomainError(t, roles.Save(ctx, dup), constant.CodeRoleCodeDuplicate)
		})
		savepoint("system_company", func() {
			_, err := tx.ExecContext(ctx, `UPDATE roles SET is_system = TRUE WHERE id = $1`, custom.ID)
			assert.Error(t, err, "CHECK: role sistem tidak boleh milik company")
		})

		mkPerm := func(solutionID, code string) *permentity.Permission {
			d, err := permentity.NewDefinition(code, code, nil)
			require.NoError(t, err)
			p := permentity.NewPermission(solutionID, d, time.Now())
			require.NoError(t, perms.Save(ctx, p))
			return p
		}
		pa, pb := mkPerm(solA, "a.read"), mkPerm(solB, "b.read")

		require.NoError(t, rolePerms.ReplaceAll(ctx, global.ID, solA, []string{pa.ID}))
		ids, err := rolePerms.PermissionIDs(ctx, global.ID)
		require.NoError(t, err)
		assert.Equal(t, map[string]bool{pa.ID: true}, ids)

		// Composite FK menolak permission solution lain.
		testhelper.AssertDomainError(t, rolePerms.ReplaceAll(ctx, global.ID, solA, []string{pb.ID}), constant.CodeRolePermissionUnknown)
	})
}
