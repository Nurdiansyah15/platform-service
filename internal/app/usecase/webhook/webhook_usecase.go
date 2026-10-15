package webhook

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	solutionrepo "platform-service/internal/domain/solution/repository"
	"platform-service/internal/domain/webhook/entity"
	webhookrepo "platform-service/internal/domain/webhook/repository"
	"platform-service/internal/domain/webhook/valueobject"
)

const defaultEventLimit = 50

type WebhookDeps struct {
	Endpoints webhookrepo.EndpointRepository
	Solutions solutionrepo.SolutionRepository
	Events    port.EventQueryReadModel
	Cipher    port.SecretCipher
	// AllowInsecureURL — izinkan URL http (dev/test).
	AllowInsecureURL bool
}

type WebhookUseCase struct {
	deps WebhookDeps
	now  func() time.Time
}

func NewWebhookUseCase(deps WebhookDeps) *WebhookUseCase {
	return &WebhookUseCase{deps: deps, now: time.Now}
}

// Create mendaftarkan endpoint webhook. Secret HMAC disimpan terenkripsi dan
// hanya dikembalikan sekali di respons ini.
//
// Required — auth: user (platform admin) | scope: -
func (uc *WebhookUseCase) Create(ctx context.Context, solutionID string, req dto.CreateWebhookRequest) (*dto.WebhookWithSecretResponse, error) {
	if _, err := uc.deps.Solutions.FindByID(ctx, solutionID); err != nil {
		return nil, mapWebhookDomainError(err)
	}
	url, err := valueobject.NewEndpointURL(req.URL, uc.deps.AllowInsecureURL)
	if err != nil {
		return nil, mapWebhookDomainError(err)
	}
	eventTypes, err := valueobject.NewEventTypes(req.EventTypes)
	if err != nil {
		return nil, mapWebhookDomainError(err)
	}
	secret, err := newSecret()
	if err != nil {
		return nil, mapWebhookDomainError(err)
	}
	encrypted, err := uc.deps.Cipher.Encrypt(secret)
	if err != nil {
		return nil, mapWebhookDomainError(err)
	}

	e := entity.NewEndpoint(solutionID, url, encrypted, eventTypes, uc.now())
	if err := uc.deps.Endpoints.Save(ctx, e); err != nil {
		return nil, mapWebhookDomainError(err)
	}
	return &dto.WebhookWithSecretResponse{WebhookResponse: toWebhookResponse(e), Secret: secret}, nil
}

// ListEvents — polling / replay event untuk solution pemanggil (aturan routing sama
// dengan webhook). Urut occurred_at naik; lanjutkan dengan meta.next_cursor.
//
// Required — auth: service | scope: events:read
func (uc *WebhookUseCase) ListEvents(ctx context.Context, actor *principal.Principal, q dto.ListEventsQuery) ([]dto.EventItem, dto.CursorMeta, error) {
	limit := defaultEventLimit
	if q.Limit != nil {
		limit = *q.Limit
	}
	rq := port.EventListReadQuery{
		SolutionID: actor.SolutionID,
		Since:      q.Since,
		EventType:  q.EventType,
		TenantID:   q.TenantID,
		Limit:      limit + 1, // satu ekstra untuk mendeteksi halaman berikutnya
	}
	if q.Cursor != nil {
		at, id, err := decodeCursor(*q.Cursor)
		if err != nil {
			return nil, dto.CursorMeta{}, mapWebhookDomainError(err)
		}
		rq.AfterOccurredAt, rq.AfterID = &at, id
	}

	rows, err := uc.deps.Events.List(ctx, rq)
	if err != nil {
		return nil, dto.CursorMeta{}, mapWebhookDomainError(err)
	}
	meta := dto.CursorMeta{Limit: limit}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1]
		next := encodeCursor(last.OccurredAt, last.ID)
		meta.NextCursor = &next
	}
	items := make([]dto.EventItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, toEventItem(r.ID, r.EventType, r.TenantID, r.SolutionID, r.OccurredAt, r.Payload))
	}
	return items, meta, nil
}

func toEventItem(id, eventType string, tenantID, solutionID *string, occurredAt time.Time, payload []byte) dto.EventItem {
	return dto.EventItem{
		ID: id, Type: eventType, TenantID: tenantID, SolutionID: solutionID,
		OccurredAt: occurredAt.UTC(), Data: payload,
	}
}
