package seeders_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/seeders"
	"platform-service/internal/testhelper"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	var cleanup func()
	testDB, cleanup = testhelper.MustStartTestDB()
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func TestPlatformAdminSeeder_Idempotent(t *testing.T) {
	ctx := context.Background()
	email := testhelper.UniqueEmail("seed-admin")
	s := seeders.PlatformAdminSeeder{Email: email, Password: "Secret123!", Hasher: testhelper.TestPasswordHasher()}

	require.NoError(t, s.Run(ctx, testDB))
	var firstHash string
	require.NoError(t, testDB.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE email = $1`, email).Scan(&firstHash))

	// Jalankan ulang dengan password berbeda: user tidak diduplikasi, password tidak ditimpa.
	s.Password = "PasswordLain123!"
	require.NoError(t, s.Run(ctx, testDB))

	var (
		count    int
		hash     string
		isAdmin  bool
		userName string
	)
	require.NoError(t, testDB.QueryRowContext(ctx,
		`SELECT COUNT(*) OVER (), password_hash, is_platform_admin, name FROM users WHERE email = $1`, email,
	).Scan(&count, &hash, &isAdmin, &userName))
	assert.Equal(t, 1, count)
	assert.Equal(t, firstHash, hash)
	assert.True(t, isAdmin)
	assert.Equal(t, "Platform Admin", userName)
}

func TestPlatformAdminSeeder_PromotesExistingUser(t *testing.T) {
	ctx := context.Background()
	email := testhelper.UniqueEmail("existing")
	_, err := testDB.ExecContext(ctx, `INSERT INTO users (email, name) VALUES ($1, 'Lama')`, email)
	require.NoError(t, err)

	require.NoError(t, seeders.PlatformAdminSeeder{Email: email, Password: "Secret123!", Hasher: testhelper.TestPasswordHasher()}.Run(ctx, testDB))

	var isAdmin bool
	require.NoError(t, testDB.QueryRowContext(ctx, `SELECT is_platform_admin FROM users WHERE email = $1`, email).Scan(&isAdmin))
	assert.True(t, isAdmin)
}

func TestPlatformAdminSeeder_RequiresCredentials(t *testing.T) {
	err := seeders.PlatformAdminSeeder{}.Run(context.Background(), testDB)
	assert.ErrorContains(t, err, "PLATFORM_ADMIN_EMAIL")
}
