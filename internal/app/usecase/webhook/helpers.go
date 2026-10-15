package webhook

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	domainerr "platform-service/internal/domain/errors"
	solutionconstant "platform-service/internal/domain/solution/constant"
	"platform-service/internal/domain/webhook/constant"
	"platform-service/internal/domain/webhook/entity"
)

const (
	secretPrefix = "whsec_"
	secretBytes  = 32
)

func newSecret() (string, error) {
	b := make([]byte, secretBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("webhook: generate secret: %w", err)
	}
	return secretPrefix + base64.RawURLEncoding.EncodeToString(b), nil
}

func mapWebhookDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case constant.CodeWebhookNotFound, solutionconstant.CodeSolutionNotFound:
		return apperror.NotFound(string(de.Code))
	case constant.CodeWebhookURLInvalid, constant.CodeWebhookEventTypeInvalid, constant.CodeEventCursorInvalid:
		return apperror.Unprocessable(string(de.Code), nil, err)
	default:
		return apperror.Internal(string(de.Code), err)
	}
}

func toWebhookResponse(e *entity.Endpoint) dto.WebhookResponse {
	return dto.WebhookResponse{
		ID: e.ID, SolutionID: e.SolutionID, URL: e.URL, EventTypes: e.EventTypes,
		Status: string(e.Status), CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	}
}

// cursor = base64url("<occurred_at RFC3339Nano>|<event id>").
func encodeCursor(occurredAt time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(occurredAt.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func decodeCursor(raw string) (time.Time, string, error) {
	invalid := domainerr.New(constant.CodeEventCursorInvalid)
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return time.Time{}, "", invalid
	}
	ts, id, ok := strings.Cut(string(b), "|")
	if !ok {
		return time.Time{}, "", invalid
	}
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return time.Time{}, "", invalid
	}
	if _, err := uuid.Parse(id); err != nil {
		return time.Time{}, "", invalid
	}
	return t, id, nil
}
