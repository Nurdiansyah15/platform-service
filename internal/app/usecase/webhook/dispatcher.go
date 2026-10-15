package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"platform-service/internal/app/port"
	"platform-service/internal/domain/webhook/entity"
	webhookrepo "platform-service/internal/domain/webhook/repository"
	"platform-service/internal/domain/webhook/service"
)

// Header pengiriman webhook (spec §2.9).
const (
	HeaderEventID   = "X-Platform-Event-Id"
	HeaderEventType = "X-Platform-Event-Type"
	HeaderTimestamp = "X-Platform-Timestamp"
	HeaderSignature = "X-Platform-Signature"
)

type DispatcherDeps struct {
	Deliveries webhookrepo.DeliveryRepository
	Sender     port.WebhookSender
	Cipher     port.SecretCipher
	Logger     *slog.Logger
	BatchSize  int
	// Lease — lama delivery yang sedang dikirim disembunyikan dari worker lain.
	// Harus lebih lama dari timeout HTTP sender.
	Lease time.Duration
}

// Dispatcher memindahkan event outbox ke webhook_deliveries (fan-out) lalu
// mengirim delivery yang jatuh tempo. Dijalankan oleh cmd/worker.
type Dispatcher struct {
	deps DispatcherDeps
	now  func() time.Time
}

func NewDispatcher(deps DispatcherDeps) *Dispatcher {
	if deps.BatchSize <= 0 {
		deps.BatchSize = 50
	}
	if deps.Lease <= 0 {
		deps.Lease = time.Minute
	}
	return &Dispatcher{deps: deps, now: time.Now}
}

// SetClock mengganti sumber waktu (test).
func (d *Dispatcher) SetClock(now func() time.Time) { d.now = now }

// RunOnce menjalankan satu siklus: fan-out lalu kirim delivery jatuh tempo.
func (d *Dispatcher) RunOnce(ctx context.Context) (created, sent int, err error) {
	if created, err = d.deps.Deliveries.FanOut(ctx, d.now(), d.deps.BatchSize*10); err != nil {
		return 0, 0, err
	}
	sent, err = d.DeliverDue(ctx)
	return created, sent, err
}

// DeliverDue mengirim delivery pending/failed yang next_attempt_at-nya sudah lewat.
func (d *Dispatcher) DeliverDue(ctx context.Context) (int, error) {
	due, err := d.deps.Deliveries.ClaimDue(ctx, d.now(), d.deps.Lease, d.deps.BatchSize)
	if err != nil {
		return 0, err
	}
	for _, dl := range due {
		statusCode, sendErr := d.send(ctx, dl)
		outcome := dl.Evaluate(statusCode, sendErr, d.now())
		if err := d.deps.Deliveries.SaveOutcome(ctx, dl.ID, outcome); err != nil {
			return 0, err
		}
		if outcome.Error != nil {
			d.deps.Logger.Warn("webhook delivery failed",
				"delivery_id", dl.ID, "event_id", dl.EventID, "attempt", outcome.AttemptCount,
				"status", string(outcome.Status), "error", *outcome.Error)
		}
	}
	return len(due), nil
}

func (d *Dispatcher) send(ctx context.Context, dl *entity.Delivery) (int, error) {
	secret, err := d.deps.Cipher.Decrypt(dl.SecretEncrypted)
	if err != nil {
		return 0, fmt.Errorf("webhook: dekripsi secret: %w", err)
	}
	body, err := json.Marshal(toEventItem(dl.EventID, dl.EventType, dl.TenantID, dl.SolutionID, dl.OccurredAt, dl.Payload))
	if err != nil {
		return 0, fmt.Errorf("webhook: marshal body: %w", err)
	}
	ts := d.now().Unix()
	return d.deps.Sender.Send(ctx, port.WebhookRequest{
		URL: dl.URL,
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"User-Agent":    "platform-webhooks/1",
			HeaderEventID:   dl.EventID,
			HeaderEventType: dl.EventType,
			HeaderTimestamp: strconv.FormatInt(ts, 10),
			HeaderSignature: service.Sign(secret, ts, body),
		},
		Body: body,
	})
}
