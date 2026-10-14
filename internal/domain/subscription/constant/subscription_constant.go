package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodeSubscriptionNotFound          domainerr.Code = "DOMAIN_SUBSCRIPTION_NOT_FOUND"
	CodeSubscriptionAlreadyLive       domainerr.Code = "DOMAIN_SUBSCRIPTION_ALREADY_LIVE"
	CodeSubscriptionNotLive           domainerr.Code = "DOMAIN_SUBSCRIPTION_NOT_LIVE"
	CodeSubscriptionStatusInvalid     domainerr.Code = "DOMAIN_SUBSCRIPTION_STATUS_INVALID"
	CodeSubscriptionSamePlan          domainerr.Code = "DOMAIN_SUBSCRIPTION_SAME_PLAN"
	CodeSubscriptionPlanUnavailable   domainerr.Code = "DOMAIN_SUBSCRIPTION_PLAN_UNAVAILABLE"
	CodeSubscriptionTenantInactive    domainerr.Code = "DOMAIN_SUBSCRIPTION_TENANT_INACTIVE"
	CodeSubscriptionPersistenceFailed domainerr.Code = "DOMAIN_SUBSCRIPTION_PERSISTENCE_FAILED"
	CodeSubscriptionQueryFailed       domainerr.Code = "DOMAIN_SUBSCRIPTION_QUERY_FAILED"

	CodeSubscriptionAddonNotFound      domainerr.Code = "DOMAIN_SUBSCRIPTION_ADDON_NOT_FOUND"
	CodeSubscriptionAddonAlreadyActive domainerr.Code = "DOMAIN_SUBSCRIPTION_ADDON_ALREADY_ACTIVE"
	CodeSubscriptionAddonUnavailable   domainerr.Code = "DOMAIN_SUBSCRIPTION_ADDON_UNAVAILABLE"
	CodeSubscriptionAddonQuantity      domainerr.Code = "DOMAIN_SUBSCRIPTION_ADDON_QUANTITY_INVALID"
)
