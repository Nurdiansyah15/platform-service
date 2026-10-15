package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/webhook/entity"
	"platform-service/internal/domain/webhook/valueobject"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

// seedWebhookDelivery membuat endpoint + satu event milik solution, lalu fan-out.
func seedWebhookDelivery(t *testing.T) (endpointID string) {
	t.Helper()
	ctx := context.Background()
	solutionID := testhelper.MustSeedSolution(t, testDB, testhelper.UniqueCode("wh"), "active")
	ep := entity.NewEndpoint(solutionID, "http://127.0.0.1:1/hook", "enc", []string{"role.created"}, time.Now().Add(-time.Minute))
	require.NoError(t, persistence.NewPostgresWebhookEndpointRepository(testDB).Save(ctx, ep))
	_, err := testDB.ExecContext(ctx, `
		INSERT INTO events (id, event_type, solution_id, payload, occurred_at) VALUES ($1, 'role.created', $2, '{}', NOW())`,
		uuid.NewString(), solutionID)
	require.NoError(t, err)
	_, err = persistence.NewPostgresWebhookDeliveryRepository(testDB).FanOut(ctx, time.Now(), 1000)
	require.NoError(t, err)
	return ep.ID
}

func claimedFor(ds []*entity.Delivery, endpointID string) []*entity.Delivery {
	var out []*entity.Delivery
	for _, d := range ds {
		if d.EndpointID == endpointID {
			out = append(out, d)
		}
	}
	return out
}

func TestWebhookDeliveryRepository_FanOutIsIdempotent(t *testing.T) {
	endpointID := seedWebhookDelivery(t)
	_, err := persistence.NewPostgresWebhookDeliveryRepository(testDB).FanOut(context.Background(), time.Now(), 1000)
	require.NoError(t, err)

	var n int
	require.NoError(t, testDB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM webhook_deliveries WHERE webhook_endpoint_id = $1`, endpointID).Scan(&n))
	assert.Equal(t, 1, n)
}

func TestWebhookDeliveryRepository_ClaimDueLease(t *testing.T) {
	ctx := context.Background()
	repo := persistence.NewPostgresWebhookDeliveryRepository(testDB)
	endpointID := seedWebhookDelivery(t)
	now := time.Now()

	first, err := repo.ClaimDue(ctx, now, time.Minute, 1000)
	require.NoError(t, err)
	claimed := claimedFor(first, endpointID)
	require.Len(t, claimed, 1)
	d := claimed[0]
	assert.Equal(t, "role.created", d.EventType)
	assert.Equal(t, "http://127.0.0.1:1/hook", d.URL)
	assert.Equal(t, "enc", d.SecretEncrypted)
	assert.Equal(t, 0, d.AttemptCount)

	// Selama lease, worker lain tidak mendapat delivery yang sama.
	second, err := repo.ClaimDue(ctx, now, time.Minute, 1000)
	require.NoError(t, err)
	assert.Empty(t, claimedFor(second, endpointID))

	// Worker mati sebelum menyimpan hasil → lease habis → bisa diklaim ulang.
	again, err := repo.ClaimDue(ctx, now.Add(2*time.Minute), time.Minute, 1000)
	require.NoError(t, err)
	assert.Len(t, claimedFor(again, endpointID), 1)

	// Hasil final → tidak diklaim lagi.
	delivered := now
	require.NoError(t, repo.SaveOutcome(ctx, d.ID, entity.Outcome{
		Status: valueobject.DeliverySuccess, AttemptCount: 1, DeliveredAt: &delivered,
	}))
	later, err := repo.ClaimDue(ctx, now.Add(time.Hour), time.Minute, 1000)
	require.NoError(t, err)
	assert.Empty(t, claimedFor(later, endpointID))
}

func TestWebhookDeliveryRepository_DisabledEndpointNotClaimed(t *testing.T) {
	ctx := context.Background()
	endpointID := seedWebhookDelivery(t)
	_, err := testDB.ExecContext(ctx, `UPDATE webhook_endpoints SET status = 'disabled' WHERE id = $1`, endpointID)
	require.NoError(t, err)

	got, err := persistence.NewPostgresWebhookDeliveryRepository(testDB).ClaimDue(ctx, time.Now(), time.Minute, 1000)
	require.NoError(t, err)
	assert.Empty(t, claimedFor(got, endpointID))
}
