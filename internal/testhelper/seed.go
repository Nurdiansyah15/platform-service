package testhelper

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// UniqueCode menghasilkan code unik huruf kecil (aman untuk code solution/plan/dll).
func UniqueCode(prefix string) string {
	return prefix + "-" + uuid.NewString()[:8]
}

// MustSeedSolution menyisipkan solution langsung ke DB dan mengembalikan ID-nya.
func MustSeedSolution(t *testing.T, db *sql.DB, code, status string) string {
	t.Helper()
	var id string
	err := db.QueryRowContext(context.Background(),
		`INSERT INTO solutions (code, name, status) VALUES ($1, $2, $3) RETURNING id`,
		code, "Solution "+code, status,
	).Scan(&id)
	require.NoError(t, err, "MustSeedSolution")
	return id
}
