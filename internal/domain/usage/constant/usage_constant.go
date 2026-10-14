package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodeUsageQuantityInvalid       domainerr.Code = "DOMAIN_USAGE_QUANTITY_INVALID"
	CodeUsageIdempotencyKeyInvalid domainerr.Code = "DOMAIN_USAGE_IDEMPOTENCY_KEY_INVALID"
	CodeUsageNotMetered            domainerr.Code = "DOMAIN_USAGE_NOT_METERED"
	CodeUsageWrongMode             domainerr.Code = "DOMAIN_USAGE_WRONG_MODE"
	CodeUsageSolutionMismatch      domainerr.Code = "DOMAIN_USAGE_SOLUTION_MISMATCH"
	CodeUsageNoSubscription        domainerr.Code = "DOMAIN_USAGE_NO_SUBSCRIPTION"
	CodeUsageDuplicateKey          domainerr.Code = "DOMAIN_USAGE_DUPLICATE_KEY"
	CodeUsageIdempotencyConflict   domainerr.Code = "DOMAIN_USAGE_IDEMPOTENCY_CONFLICT"
	CodeUsageRecordNotFound        domainerr.Code = "DOMAIN_USAGE_RECORD_NOT_FOUND"
	CodeUsagePersistenceFailed     domainerr.Code = "DOMAIN_USAGE_PERSISTENCE_FAILED"
	CodeUsageQueryFailed           domainerr.Code = "DOMAIN_USAGE_QUERY_FAILED"
)

const IdempotencyKeyMaxLength = 150
