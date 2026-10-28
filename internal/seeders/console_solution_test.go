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
