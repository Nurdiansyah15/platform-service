package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/authz/constant"
	"platform-service/internal/domain/authz/entity"
	companyentity "platform-service/internal/domain/organization/entity"
	roleentity "platform-service/internal/domain/role/entity"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func TestPostgresRoleAssignmentRepository(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		tx := persistence.TxFromContext(ctx)
		solID := seedSolutionTx(t, ctx)
		tn, err := companyentity.NewCompany("A", testhelper.UniqueCode("a"), time.Now())
		require.NoError(t, err)
		require.NoError(t, persistence.NewPostgresCompanyRepository(testDB).Save(ctx, tn))
		member := mustSaveUser(t, ctx, testhelper.UniqueEmail("m"))
		m, err := companyentity.NewCompanyMember(tn.ID, member.ID, "", time.Now())
		require.NoError(t, err)
		require.NoError(t, persistence.NewPostgresCompanyMemberRepository(testDB).Save(ctx, m))
		role, err := roleentity.NewRole(roleentity.NewRoleParams{SolutionID: solID, Code: "r", Name: "R"}, time.Now())
		require.NoError(t, err)
		require.NoError(t, persistence.NewPostgresRoleRepository(testDB).Save(ctx, role))

		repo := persistence.NewPostgresRoleAssignmentRepository(testDB)
		scope := map[string]string{"branch": "b-1"}
		require.NoError(t, repo.Create(ctx, entity.NewRoleAssignment(tn.ID, member.ID, role.ID, scope, nil, time.Now())))

		_, err = tx.ExecContext(ctx, `SAVEPOINT dup`)
		require.NoError(t, err)
		testhelper.AssertDomainError(t, repo.Create(ctx, entity.NewRoleAssignment(tn.ID, member.ID, role.ID, scope, nil, time.Now())),
			constant.CodeAssignmentDuplicate)
		_, err = tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT dup`)
		require.NoError(t, err)

		// FK (company_id, user_id) → company_members: bukan member ditolak DB.
		// Role sama dengan nilai scope berbeda = assignment lain.
		require.NoError(t, repo.Create(ctx, entity.NewRoleAssignment(tn.ID, member.ID, role.ID, map[string]string{"branch": "b-2"}, nil, time.Now())))
		require.NoError(t, repo.DeleteByScope(ctx, "branch", "b-1"))
		var n int
		require.NoError(t, tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_role_assignments WHERE role_id = $1`, role.ID).Scan(&n))
		assert.Equal(t, 1, n, "hanya assignment ber-nilai branch b-1 yang terhapus")

		testhelper.AssertDomainError(t, repo.Create(ctx, entity.NewRoleAssignment(tn.ID, uuid.NewString(), role.ID, scope, nil, time.Now())),
			constant.CodeAssignmentNotMember)
	})
}
