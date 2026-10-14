package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"platform-service/internal/app/port"
)

// ErrOutboxRequiresTx dikembalikan bila Append dipanggil di luar Transactor.WithTx.
var ErrOutboxRequiresTx = errors.New("outbox: Append harus dipanggil di dalam transaksi")

// PostgresEventOutbox implements port.EventOutbox.
type PostgresEventOutbox struct{}

func NewPostgresEventOutbox() *PostgresEventOutbox {
	return &PostgresEventOutbox{}
}

func (o *PostgresEventOutbox) Append(ctx context.Context, events ...port.OutboxEvent) error {
	tx := TxFromContext(ctx)
	if tx == nil {
		return ErrOutboxRequiresTx
	}

	const q = `
		INSERT INTO events (id, event_type, tenant_id, solution_id, payload, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	for _, ev := range events {
		if ev.EventType == "" {
			return errors.New("outbox: event_type wajib diisi")
		}
		payload, err := json.Marshal(ev.Payload)
		if err != nil {
			return fmt.Errorf("outbox: marshal payload %s: %w", ev.EventType, err)
		}
		id := ev.ID
		if id == "" {
			id = uuid.NewString()
		}
		occurredAt := ev.OccurredAt
		if occurredAt.IsZero() {
			occurredAt = time.Now().UTC()
		}
		if _, err := tx.ExecContext(ctx, q, id, ev.EventType, nullString(ev.TenantID), nullString(ev.SolutionID), payload, occurredAt); err != nil {
			return fmt.Errorf("outbox: insert %s: %w", ev.EventType, err)
		}
	}
	return nil
}

func nullString(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}
