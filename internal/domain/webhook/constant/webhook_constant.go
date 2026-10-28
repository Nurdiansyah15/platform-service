package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodeWebhookNotFound          domainerr.Code = "DOMAIN_WEBHOOK_NOT_FOUND"
	CodeWebhookURLInvalid        domainerr.Code = "DOMAIN_WEBHOOK_URL_INVALID"
	CodeWebhookEventTypeInvalid  domainerr.Code = "DOMAIN_WEBHOOK_EVENT_TYPE_INVALID"
	CodeWebhookPersistenceFailed domainerr.Code = "DOMAIN_WEBHOOK_PERSISTENCE_FAILED"
	CodeWebhookQueryFailed       domainerr.Code = "DOMAIN_WEBHOOK_QUERY_FAILED"
	CodeEventCursorInvalid       domainerr.Code = "DOMAIN_EVENT_CURSOR_INVALID"
)

// EventTypes adalah katalog event (spec §2.9). Endpoint hanya boleh berlangganan event ini.
var EventTypes = []string{
	"user.updated", "user.status_changed",
	"company.created", "company.updated", "company.deleted",
	"company_member.added", "company_member.updated", "company_member.removed",
	"branch.created", "branch.updated", "branch.deleted",
	"subscription.created", "subscription.plan_changed", "subscription.status_changed",
	"subscription_addon.added", "subscription_addon.updated", "subscription_addon.removed",
	"entitlement.changed",
	"role.created", "role.updated", "role.deleted", "role.permissions_changed", "role_assignment.created", "role_assignment.deleted",
	"usage.limit_reached",
	"license.installed", "license.state_changed", "license.expiring",
}

// Kebijakan retry pengiriman.
const (
	MaxDeliveryAttempts = 8
	ResponseExcerptMax  = 500
)
