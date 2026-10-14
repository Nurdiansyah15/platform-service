package persistence

import (
	"context"
	"database/sql"
)

type txKey struct{}

// PostgresTransactor implements port.Transactor.
type PostgresTransactor struct {
	db *sql.DB
}

func NewPostgresTransactor(db *sql.DB) *PostgresTransactor {
	return &PostgresTransactor{db: db}
}

func (t *PostgresTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	txCtx := context.WithValue(ctx, txKey{}, tx)

	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// TxFromContext returns the *sql.Tx stored in ctx by WithTx, or nil if none.
func TxFromContext(ctx context.Context) *sql.Tx {
	tx, _ := ctx.Value(txKey{}).(*sql.Tx)
	return tx
}

// NewTxContext returns a new context carrying tx. Used by test helpers to inject
// a transaction so that execFromContext picks it up without exposing txKey.
func NewTxContext(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// executor is a minimal interface satisfied by both *sql.DB and *sql.Tx.
type executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// execFromContext returns the active *sql.Tx if one exists, otherwise falls
// back to the *sql.DB. Repos that need TX-awareness call this instead of r.db.
func execFromContext(ctx context.Context, db *sql.DB) executor {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return db
}
