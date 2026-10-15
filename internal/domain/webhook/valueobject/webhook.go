package valueobject

import (
	"net/url"
	"slices"
	"strings"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/webhook/constant"
)

// NewEndpointURL memvalidasi URL webhook: absolut, host terisi, https (http hanya bila allowInsecure).
func NewEndpointURL(raw string, allowInsecure bool) (string, error) {
	v := strings.TrimSpace(raw)
	u, err := url.Parse(v)
	if err != nil || u.Host == "" || len(v) > 2048 {
		return "", domainerr.New(constant.CodeWebhookURLInvalid)
	}
	insecureAllowed := allowInsecure && u.Scheme == "http"
	if u.Scheme != "https" && !insecureAllowed {
		return "", domainerr.New(constant.CodeWebhookURLInvalid)
	}
	return v, nil
}

// NewEventTypes memvalidasi daftar event terhadap katalog; hasil terurut & unik.
func NewEventTypes(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, domainerr.New(constant.CodeWebhookEventTypeInvalid)
	}
	out := make([]string, 0, len(raw))
	for _, e := range raw {
		e = strings.TrimSpace(e)
		if !slices.Contains(constant.EventTypes, e) {
			return nil, domainerr.New(constant.CodeWebhookEventTypeInvalid)
		}
		out = append(out, e)
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}

type EndpointStatus string

const (
	EndpointActive   EndpointStatus = "active"
	EndpointDisabled EndpointStatus = "disabled"
)

type DeliveryStatus string

const (
	DeliveryPending DeliveryStatus = "pending"
	DeliverySuccess DeliveryStatus = "success"
	DeliveryFailed  DeliveryStatus = "failed"
	DeliveryDead    DeliveryStatus = "dead"
)
