package persistence_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/app/port"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func TestPostgresEventOutbox_Append_RequiresTx(t *testing.T) {
	err := persistence.NewPostgresEventOutbox().Append(context.Background(),
		port.OutboxEvent{EventType: "company.updated", Payload: map[string]any{}})

	require.ErrorIs(t, err, persistence.ErrOutboxRequiresTx)
}

func TestPostgresEventOutbox_Append_PersistsAllFields(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		id := uuid.NewString()
		companyID := uuid.NewString()
		occurredAt := time.Date(2026, 10, 14, 7, 0, 0, 0, time.UTC)

		err := persistence.NewPostgresEventOutbox().Append(ctx, port.OutboxEvent{
			ID:         id,
			EventType:  "company.updated",
			CompanyID:  &companyID,
			Payload:    map[string]any{"name": "Acme"},
			OccurredAt: occurredAt,
		})
		require.NoError(t, err)

		var (
			gotType     string
			gotCompany  *string
			gotSolution *string
			gotPayload  []byte
			gotOccurred time.Time
		)
		tx := persistence.TxFromContext(ctx)
		require.NoError(t, tx.QueryRowContext(ctx,
			`SELECT event_type, company_id, solution_id, payload, occurred_at FROM events WHERE id = $1`, id,
		).Scan(&gotType, &gotCompany, &gotSolution, &gotPayload, &gotOccurred))

		assert.Equal(t, "company.updated", gotType)
		require.NotNil(t, gotCompany)
		assert.Equal(t, companyID, *gotCompany)
		assert.Nil(t, gotSolution)
		assert.True(t, occurredAt.Equal(gotOccurred))

		var payload map[string]any
		require.NoError(t, json.Unmarshal(gotPayload, &payload))
		assert.Equal(t, "Acme", payload["name"])
	})
}

func TestPostgresEventOutbox_Append_GeneratesIDAndTimestamp(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		eventType := "test.defaults." + uuid.NewString()
		require.NoError(t, persistence.NewPostgresEventOutbox().Append(ctx,
			port.OutboxEvent{EventType: eventType, Payload: map[string]any{}},
			port.OutboxEvent{EventType: eventType, Payload: map[string]any{}},
		))

		var n int
		var latest time.Time
		require.NoError(t, persistence.TxFromContext(ctx).QueryRowContext(ctx,
			`SELECT COUNT(DISTINCT id), MAX(occurred_at) FROM events WHERE event_type = $1`, eventType,
		).Scan(&n, &latest))
		assert.Equal(t, 2, n)
		assert.WithinDuration(t, time.Now(), latest, time.Minute)
	})
}

func TestPostgresEventOutbox_Append_RejectsEmptyType(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		err := persistence.NewPostgresEventOutbox().Append(ctx, port.OutboxEvent{Payload: map[string]any{}})
		assert.Error(t, err)
	})
}
