package entity

import (
	"time"

	"github.com/google/uuid"

	"platform-service/internal/domain/webhook/constant"
	"platform-service/internal/domain/webhook/valueobject"
)

type Endpoint struct {
	ID              string
	SolutionID      string
	URL             string
	SecretEncrypted string
	EventTypes      []string
	Status          valueobject.EndpointStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewEndpoint(solutionID, url, secretEncrypted string, eventTypes []string, now time.Time) *Endpoint {
	now = now.UTC()
	return &Endpoint{
		ID: uuid.NewString(), SolutionID: solutionID, URL: url, SecretEncrypted: secretEncrypted,
		EventTypes: eventTypes, Status: valueobject.EndpointActive, CreatedAt: now, UpdatedAt: now,
	}
}

// Delivery adalah satu percobaan pengiriman event ke endpoint (beserta data untuk mengirim).
type Delivery struct {
	ID              string
	EventID         string
	EndpointID      string
	AttemptCount    int
	EventType       string
	TenantID        *string
	SolutionID      *string
	Payload         []byte
	OccurredAt      time.Time
	URL             string
	SecretEncrypted string
}

// Outcome adalah hasil satu percobaan pengiriman.
type Outcome struct {
	Status        valueobject.DeliveryStatus
	AttemptCount  int
	NextAttemptAt *time.Time
	ResponseCode  *int
	Error         *string
	DeliveredAt   *time.Time
}

// Backoff menghitung jeda sebelum percobaan berikutnya: 30 detik × 2^(n-1), maks 6 jam.
func Backoff(attempt int) time.Duration {
	const (
		base = 30 * time.Second
		max  = 6 * time.Hour
	)
	d := base
	for i := 1; i < attempt; i++ {
		d *= 2
		if d >= max {
			return max
		}
	}
	return d
}

// Evaluate menentukan status delivery setelah satu percobaan.
func (d *Delivery) Evaluate(statusCode int, sendErr error, now time.Time) Outcome {
	attempt := d.AttemptCount + 1
	now = now.UTC()
	out := Outcome{AttemptCount: attempt}
	if statusCode > 0 {
		out.ResponseCode = &statusCode
	}
	if sendErr == nil && statusCode >= 200 && statusCode < 300 {
		out.Status = valueobject.DeliverySuccess
		out.DeliveredAt = &now
		return out
	}

	msg := "non-2xx response"
	if sendErr != nil {
		msg = sendErr.Error()
	}
	if len(msg) > constant.ResponseExcerptMax {
		msg = msg[:constant.ResponseExcerptMax]
	}
	out.Error = &msg
	if attempt >= constant.MaxDeliveryAttempts {
		out.Status = valueobject.DeliveryDead
		return out
	}
	next := now.Add(Backoff(attempt))
	out.Status = valueobject.DeliveryFailed
	out.NextAttemptAt = &next
	return out
}
