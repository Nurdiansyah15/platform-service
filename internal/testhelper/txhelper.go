package testhelper

import (
	"context"
	"database/sql"
	"testing"

	"platform-service/internal/infrastructure/persistence"
)

// WithTestTx wraps fn in a transaction that is always rolled back after fn
// returns. The context passed to fn carries the *sql.Tx so that repository
// methods using execFromContext participate in the same transaction and see
// each other's writes without committing to the DB.
func WithTestTx(t *testing.T, db *sql.DB, fn func(ctx context.Context)) {
	t.Helper()

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("WithTestTx: begin transaction: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	fn(persistence.NewTxContext(context.Background(), tx))
}
