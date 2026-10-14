package port

import (
	"context"
	"time"
)

// OutboxEvent adalah satu baris tabel events (spec §2.9).
type OutboxEvent struct {
	ID         string // kosong → di-generate
	EventType  string
	TenantID   *string
	SolutionID *string
	Payload    any // di-marshal ke JSONB
	OccurredAt time.Time
}

// EventOutbox menulis event ke outbox. Append WAJIB dipanggil di dalam
// Transactor.WithTx agar event tersimpan atomik bersama perubahan datanya.
type EventOutbox interface {
	Append(ctx context.Context, events ...OutboxEvent) error
}
