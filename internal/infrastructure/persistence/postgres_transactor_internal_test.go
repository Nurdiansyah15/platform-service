package persistence

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecFromContext(t *testing.T) {
	// sql.Open tidak membuka koneksi, cukup untuk membandingkan identitas executor.
	db, err := sql.Open("pgx", "postgres://unused@localhost:1/unused")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	assert.Same(t, db, execFromContext(context.Background(), db), "tanpa tx → *sql.DB")

	tx := &sql.Tx{}
	assert.Same(t, tx, execFromContext(NewTxContext(context.Background(), tx), db), "dengan tx → *sql.Tx")
}
