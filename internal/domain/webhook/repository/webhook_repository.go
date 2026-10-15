package repository

import (
	"context"
	"time"

	"platform-service/internal/domain/webhook/entity"
)

type EndpointRepository interface {
	Save(ctx context.Context, e *entity.Endpoint) error
}

type DeliveryRepository interface {
	// FanOut membuat delivery pending untuk event yang relevan bagi endpoint active
	// (aturan routing spec §2.9). Idempotent. Mengembalikan jumlah delivery baru.
	FanOut(ctx context.Context, now time.Time, limit int) (int, error)
	// ClaimDue mengunci delivery yang jatuh tempo dengan lease (aman multi-worker).
	ClaimDue(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]*entity.Delivery, error)
	SaveOutcome(ctx context.Context, deliveryID string, o entity.Outcome) error
}
