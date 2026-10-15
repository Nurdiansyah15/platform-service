package persistence

import (
	"context"
	"database/sql"
	"time"

	domainerr "platform-service/internal/domain/errors"
	subvo "platform-service/internal/domain/subscription/valueobject"
	"platform-service/internal/domain/webhook/constant"
	"platform-service/internal/domain/webhook/entity"
)

// PostgresWebhookEndpointRepository implements repository.EndpointRepository.
type PostgresWebhookEndpointRepository struct {
	db *sql.DB
}

func NewPostgresWebhookEndpointRepository(db *sql.DB) *PostgresWebhookEndpointRepository {
	return &PostgresWebhookEndpointRepository{db: db}
}

func (r *PostgresWebhookEndpointRepository) Save(ctx context.Context, e *entity.Endpoint) error {
	_, err := execFromContext(ctx, r.db).ExecContext(ctx, `
		INSERT INTO webhook_endpoints (id, solution_id, url, secret_encrypted, event_types, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			url = EXCLUDED.url, secret_encrypted = EXCLUDED.secret_encrypted,
			event_types = EXCLUDED.event_types, status = EXCLUDED.status`,
		e.ID, e.SolutionID, e.URL, e.SecretEncrypted, e.EventTypes, string(e.Status), e.CreatedAt, e.UpdatedAt)
	if err != nil {
		return domainerr.Wrap(constant.CodeWebhookPersistenceFailed, err)
	}
	return nil
}

// PostgresWebhookDeliveryRepository implements repository.DeliveryRepository.
type PostgresWebhookDeliveryRepository struct {
	db *sql.DB
}

func NewPostgresWebhookDeliveryRepository(db *sql.DB) *PostgresWebhookDeliveryRepository {
	return &PostgresWebhookDeliveryRepository{db: db}
}

// eventRoutedToSolution — aturan routing spec §2.9 (e = events; %[1]s = solution id, %[2]s = status live):
//   - event milik solution tersebut;
//   - event tenant (solution_id NULL) dari tenant yang berlangganan (live) solution tersebut;
//   - event user (tenant & solution NULL, payload.user_id) bila user anggota aktif tenant
//     yang berlangganan solution tersebut.
const eventRoutedToSolution = `(
	e.solution_id = %[1]s
	OR (e.solution_id IS NULL AND e.tenant_id IS NOT NULL AND EXISTS (
		SELECT 1 FROM subscriptions s
		WHERE s.tenant_id = e.tenant_id AND s.solution_id = %[1]s AND s.status = ANY(%[2]s::text[])))
	OR (e.solution_id IS NULL AND e.tenant_id IS NULL AND EXISTS (
		SELECT 1 FROM tenant_members tm
		JOIN subscriptions s ON s.tenant_id = tm.tenant_id
		WHERE tm.user_id::text = e.payload->>'user_id' AND tm.status = 'active'
		  AND s.solution_id = %[1]s AND s.status = ANY(%[2]s::text[])))
)`

// liveStatuses — subscription yang masih berjalan menerima event tenant.
var liveStatuses = []string{string(subvo.StatusTrialing), string(subvo.StatusActive), string(subvo.StatusPastDue)}

func (r *PostgresWebhookDeliveryRepository) FanOut(ctx context.Context, now time.Time, limit int) (int, error) {
	res, err := execFromContext(ctx, r.db).ExecContext(ctx, `
		INSERT INTO webhook_deliveries (event_id, webhook_endpoint_id, status, next_attempt_at)
		SELECT e.id, w.id, 'pending', $1
		FROM webhook_endpoints w
		JOIN events e ON e.event_type = ANY(w.event_types) AND e.occurred_at >= w.created_at
		WHERE w.status = 'active'
		  AND `+sprintf(eventRoutedToSolution, "w.solution_id", "$2")+`
		  AND NOT EXISTS (SELECT 1 FROM webhook_deliveries d WHERE d.event_id = e.id AND d.webhook_endpoint_id = w.id)
		ORDER BY e.occurred_at
		LIMIT $3
		ON CONFLICT (event_id, webhook_endpoint_id) DO NOTHING`, now.UTC(), liveStatuses, limit)
	if err != nil {
		return 0, domainerr.Wrap(constant.CodeWebhookPersistenceFailed, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, domainerr.Wrap(constant.CodeWebhookPersistenceFailed, err)
	}
	return int(n), nil
}

func (r *PostgresWebhookDeliveryRepository) ClaimDue(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]*entity.Delivery, error) {
	// Lease: next_attempt_at dimajukan sehingga worker lain tidak mengambil delivery yang sama
	// selama pengiriman berlangsung; HTTP call dilakukan di luar transaksi.
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx, `
		WITH due AS (
			SELECT d.id FROM webhook_deliveries d
			JOIN webhook_endpoints w ON w.id = d.webhook_endpoint_id AND w.status = 'active'
			WHERE d.status IN ('pending', 'failed') AND d.next_attempt_at <= $1
			ORDER BY d.next_attempt_at
			LIMIT $3
			FOR UPDATE OF d SKIP LOCKED
		)
		UPDATE webhook_deliveries d SET next_attempt_at = $2
		FROM due, events e, webhook_endpoints w
		WHERE d.id = due.id AND e.id = d.event_id AND w.id = d.webhook_endpoint_id
		RETURNING d.id, d.event_id, d.webhook_endpoint_id, d.attempt_count, e.event_type, e.tenant_id,
			e.solution_id, e.payload, e.occurred_at, w.url, w.secret_encrypted`,
		now.UTC(), now.UTC().Add(lease), limit)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeWebhookQueryFailed, err)
	}
	defer rows.Close()

	var out []*entity.Delivery
	for rows.Next() {
		var d entity.Delivery
		if err := rows.Scan(&d.ID, &d.EventID, &d.EndpointID, &d.AttemptCount, &d.EventType, &d.TenantID,
			&d.SolutionID, &d.Payload, &d.OccurredAt, &d.URL, &d.SecretEncrypted); err != nil {
			return nil, domainerr.Wrap(constant.CodeWebhookQueryFailed, err)
		}
		out = append(out, &d)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeWebhookQueryFailed, err)
	}
	return out, nil
}

func (r *PostgresWebhookDeliveryRepository) SaveOutcome(ctx context.Context, id string, o entity.Outcome) error {
	_, err := execFromContext(ctx, r.db).ExecContext(ctx, `
		UPDATE webhook_deliveries SET status = $2, attempt_count = $3, next_attempt_at = $4,
			last_response_code = $5, last_error = $6, delivered_at = $7
		WHERE id = $1`,
		id, string(o.Status), o.AttemptCount, o.NextAttemptAt, o.ResponseCode, o.Error, o.DeliveredAt)
	if err != nil {
		return domainerr.Wrap(constant.CodeWebhookPersistenceFailed, err)
	}
	return nil
}

// PostgresEventQuery implements port.EventQueryReadModel.
type PostgresEventQuery struct {
	db *sql.DB
}

func NewPostgresEventQuery(db *sql.DB) *PostgresEventQuery {
	return &PostgresEventQuery{db: db}
}
