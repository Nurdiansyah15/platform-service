package persistence_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/app/port"
	"platform-service/internal/infrastructure/persistence"
)

func countEventsByType(t *testing.T, eventType string) int {
	t.Helper()
	var n int
	require.NoError(t, testDB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM events WHERE event_type = $1`, eventType).Scan(&n))
	return n
}

func TestPostgresTransactor_WithTx_CommitsOnSuccess(t *testing.T) {
	tr := persistence.NewPostgresTransactor(testDB)
	outbox := persistence.NewPostgresEventOutbox()
	eventType := "test.commit." + uuid.NewString()

	err := tr.WithTx(context.Background(), func(ctx context.Context) error {
		assert.NotNil(t, persistence.TxFromContext(ctx))
		return outbox.Append(ctx, port.OutboxEvent{EventType: eventType, Payload: map[string]any{"ok": true}})
	})

	require.NoError(t, err)
	assert.Equal(t, 1, countEventsByType(t, eventType))
}

func TestPostgresTransactor_WithTx_RollsBackOnError(t *testing.T) {
	tr := persistence.NewPostgresTransactor(testDB)
	outbox := persistence.NewPostgresEventOutbox()
	eventType := "test.rollback." + uuid.NewString()
	sentinel := errors.New("gagal setelah insert")

	err := tr.WithTx(context.Background(), func(ctx context.Context) error {
		require.NoError(t, outbox.Append(ctx, port.OutboxEvent{EventType: eventType, Payload: map[string]any{}}))
		return sentinel
	})

	require.ErrorIs(t, err, sentinel)
	assert.Equal(t, 0, countEventsByType(t, eventType))
}
