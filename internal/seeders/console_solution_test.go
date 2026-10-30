package seeders_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/seeders"
	"platform-service/internal/testhelper"
)

func TestConsoleSolutionSeeder_Idempotent(t *testing.T) {
	ctx := context.Background()
	code := testhelper.UniqueCode("console")
	s := seeders.ConsoleSolutionSeeder{Code: code, SolutionName: "Platform Console"}

	require.NoError(t, s.Run(ctx, testDB))
	require.NoError(t, s.Run(ctx, testDB))

	var (
		count  int
		status string
		name   string
	)
	require.NoError(t, testDB.QueryRowContext(ctx,
		`SELECT COUNT(*) OVER (), status, name FROM solutions WHERE code = $1`, code,
	).Scan(&count, &status, &name))
	assert.Equal(t, 1, count)
	assert.Equal(t, "active", status)
	assert.Equal(t, "Platform Console", name)
}

func TestConsoleSolutionSeeder_RequiresCode(t *testing.T) {
	err := seeders.ConsoleSolutionSeeder{}.Run(context.Background(), testDB)
	assert.ErrorContains(t, err, "CONSOLE_SOLUTION_CODE")
}

func TestConsoleSolutionSeeder_PlatformCatalogAndAdminRole(t *testing.T) {
	ctx := context.Background()
	code := testhelper.UniqueCode("console")
	require.NoError(t, seeders.ConsoleSolutionSeeder{Code: code}.Run(ctx, testDB))
	require.NoError(t, seeders.ConsoleSolutionSeeder{Code: code}.Run(ctx, testDB))

	var perms, rolePerms int
	var isSystem bool
	require.NoError(t, testDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM permissions p JOIN solutions s ON s.id = p.solution_id WHERE s.code = $1`, code).Scan(&perms))
	assert.Equal(t, 5, perms)
	require.NoError(t, testDB.QueryRowContext(ctx, `
		SELECT r.is_system, (SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = r.id)
		FROM roles r JOIN solutions s ON s.id = r.solution_id
		WHERE s.code = $1 AND r.code = 'company_admin' AND r.company_id IS NULL`, code).Scan(&isSystem, &rolePerms))
	assert.True(t, isSystem)
	assert.Equal(t, 5, rolePerms)
}

func TestConsoleSolutionSeeder_BackfillOnlyOnFirstInstall(t *testing.T) {
	ctx := context.Background()
	var userID, companyID string
	require.NoError(t, testDB.QueryRowContext(ctx,
		`INSERT INTO users (email, name) VALUES ($1, 'Owner') RETURNING id`, testhelper.UniqueEmail("owner")).Scan(&userID))
	require.NoError(t, testDB.QueryRowContext(ctx,
		`INSERT INTO companies (name, slug) VALUES ('Lama', $1) RETURNING id`, testhelper.UniqueCode("lama")).Scan(&companyID))
	_, err := testDB.ExecContext(ctx, `INSERT INTO company_members (company_id, user_id) VALUES ($1, $2)`, companyID, userID)
	require.NoError(t, err)

	code := testhelper.UniqueCode("console")
	require.NoError(t, seeders.ConsoleSolutionSeeder{Code: code}.Run(ctx, testDB))

	countAdmin := func() int {
		var n int
		require.NoError(t, testDB.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM user_role_assignments a
			JOIN roles r ON r.id = a.role_id JOIN solutions s ON s.id = r.solution_id
			WHERE s.code = $1 AND r.code = 'company_admin' AND a.user_id = $2 AND a.scope_id = $3`,
			code, userID, companyID).Scan(&n))
		return n
	}
	assert.Equal(t, 1, countAdmin(), "member company lama menjadi admin saat katalog dipasang")

	_, err = testDB.ExecContext(ctx, `
		DELETE FROM user_role_assignments WHERE user_id = $1 AND company_id = $2`, userID, companyID)
	require.NoError(t, err)
	require.NoError(t, seeders.ConsoleSolutionSeeder{Code: code}.Run(ctx, testDB))
	assert.Equal(t, 0, countAdmin(), "seed ulang tidak memberi admin lagi")
}
