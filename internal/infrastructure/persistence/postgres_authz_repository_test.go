package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/authz/constant"
	"platform-service/internal/domain/authz/entity"
	"platform-service/internal/domain/authz/valueobject"
	roleentity "platform-service/internal/domain/role/entity"
	tenantentity "platform-service/internal/domain/tenant/entity"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func TestPostgresRoleAssignmentRepository(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		tx := persistence.TxFromContext(ctx)
		solID := seedSolutionTx(t, ctx)
		tn, err := tenantentity.NewTenant("A", testhelper.UniqueCode("a"), time.Now())
		require.NoError(t, err)
		require.NoError(t, persistence.NewPostgresTenantRepository(testDB).Save(ctx, tn))
		member := mustSaveUser(t, ctx, testhelper.UniqueEmail("m"))
		m, err := tenantentity.NewTenantMember(tn.ID, member.ID, "", time.Now())
		require.NoError(t, err)
		require.NoError(t, persistence.NewPostgresTenantMemberRepository(testDB).Save(ctx, m))
		role, err := roleentity.NewRole(roleentity.NewRoleParams{SolutionID: solID, Code: "r", Name: "R"}, time.Now())
		require.NoError(t, err)
		require.NoError(t, persistence.NewPostgresRoleRepository(testDB).Save(ctx, role))

		repo := persistence.NewPostgresRoleAssignmentRepository(testDB)
		scope := valueobject.Scope{Type: valueobject.ScopeTenant, ID: tn.ID}
		require.NoError(t, repo.Create(ctx, entity.NewRoleAssignment(tn.ID, member.ID, role.ID, scope, time.Now())))

		_, err = tx.ExecContext(ctx, `SAVEPOINT dup`)
		require.NoError(t, err)
		testhelper.AssertDomainError(t, repo.Create(ctx, entity.NewRoleAssignment(tn.ID, member.ID, role.ID, scope, time.Now())),
			constant.CodeAssignmentDuplicate)
		_, err = tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT dup`)
		require.NoError(t, err)

		// FK (tenant_id, user_id) → tenant_members: bukan member ditolak DB.
		testhelper.AssertDomainError(t, repo.Create(ctx, entity.NewRoleAssignment(tn.ID, uuid.NewString(), role.ID, scope, time.Now())),
			constant.CodeAssignmentNotMember)
	})
}
