package dto

import (
	"encoding/json"
	"time"
)

type CreateWebhookRequest struct {
	URL string `json:"url" binding:"required,max=2048" example:"https://hr.example.com/hooks/platform"`
	// EventTypes — subset katalog event (spec §2.9).
	EventTypes []string `json:"event_types" binding:"required,min=1,max=50,dive,required,max=100"`
}

type WebhookResponse struct {
	ID         string    `json:"id"`
	SolutionID string    `json:"solution_id"`
	URL        string    `json:"url"`
	EventTypes []string  `json:"event_types"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// WebhookWithSecretResponse — secret hanya ditampilkan sekali saat create.
type WebhookWithSecretResponse struct {
	WebhookResponse
	// Secret — kunci HMAC-SHA256 untuk memverifikasi X-Platform-Signature.
	Secret string `json:"secret"`
}

// ListEventsQuery — GET /events. Cursor diambil dari meta.next_cursor halaman sebelumnya.
type ListEventsQuery struct {
	Since     *time.Time `form:"since" time_format:"2006-01-02T15:04:05Z07:00"`
	EventType *string    `form:"event_type" binding:"omitempty,max=100"`
	TenantID  *string    `form:"tenant_id" binding:"omitempty,uuid"`
	Cursor    *string    `form:"cursor" binding:"omitempty,max=200"`
	Limit     *int       `form:"limit" binding:"omitempty,min=1,max=200"`
}

// EventItem — bentuk event yang sama dengan body webhook.
type EventItem struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	TenantID   *string         `json:"tenant_id"`
	SolutionID *string         `json:"solution_id"`
	OccurredAt time.Time       `json:"occurred_at"`
	Data       json.RawMessage `json:"data" swaggertype:"object"`
}

// CursorMeta — pagination berbasis cursor. NextCursor nil = tidak ada halaman berikutnya.
type CursorMeta struct {
	Limit      int     `json:"limit"`
	NextCursor *string `json:"next_cursor"`
}
