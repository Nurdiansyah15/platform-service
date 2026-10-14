package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodePlanNotFound          domainerr.Code = "DOMAIN_PLAN_NOT_FOUND"
	CodePlanCodeDuplicate     domainerr.Code = "DOMAIN_PLAN_CODE_DUPLICATE"
	CodePlanCodeInvalid       domainerr.Code = "DOMAIN_PLAN_CODE_INVALID"
	CodePlanNameInvalid       domainerr.Code = "DOMAIN_PLAN_NAME_INVALID"
	CodePlanPersistenceFailed domainerr.Code = "DOMAIN_PLAN_PERSISTENCE_FAILED"
	CodePlanQueryFailed       domainerr.Code = "DOMAIN_PLAN_QUERY_FAILED"
)

const (
	CodeMaxLength = 100
	NameMaxLength = 255
)
