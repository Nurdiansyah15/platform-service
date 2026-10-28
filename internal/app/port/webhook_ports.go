package port

import (
	"context"
	"time"
)

// SecretCipher mengenkripsi secret yang harus bisa dibaca ulang (mis. secret HMAC webhook).
type SecretCipher interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// WebhookRequest adalah satu HTTP POST ke endpoint webhook.
type WebhookRequest struct {
	URL     string
	Headers map[string]string
	Body    []byte
}

// WebhookSender mengirim webhook. Mengembalikan status HTTP (0 bila gagal terhubung).
type WebhookSender interface {
	Send(ctx context.Context, req WebhookRequest) (int, error)
}

type EventListReadQuery struct {
	SolutionID string
	Since      *time.Time
	EventType  *string
	CompanyID  *string
	// After — cursor (occurred_at, id) dari halaman sebelumnya.
	AfterOccurredAt *time.Time
	AfterID         string
	Limit           int
}

type EventReadItem struct {
	ID         string
	EventType  string
	CompanyID  *string
	SolutionID *string
	Payload    []byte
	OccurredAt time.Time
}

type EventQueryReadModel interface {
	List(ctx context.Context, q EventListReadQuery) ([]EventReadItem, error)
}
