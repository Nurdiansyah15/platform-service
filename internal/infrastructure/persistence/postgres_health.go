package persistence

import (
	"context"
	"database/sql"
)

// PostgresHealthProbe implements port.DatabaseHealth.
type PostgresHealthProbe struct {
	db *sql.DB
}

func NewPostgresHealthProbe(db *sql.DB) *PostgresHealthProbe {
	return &PostgresHealthProbe{db: db}
}

func (p *PostgresHealthProbe) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}
