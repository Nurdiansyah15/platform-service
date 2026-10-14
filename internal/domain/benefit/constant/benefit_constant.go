package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodeBenefitNotFound           domainerr.Code = "DOMAIN_BENEFIT_NOT_FOUND"
	CodeBenefitCodeDuplicate      domainerr.Code = "DOMAIN_BENEFIT_CODE_DUPLICATE"
	CodeBenefitCodeInvalid        domainerr.Code = "DOMAIN_BENEFIT_CODE_INVALID"
	CodeBenefitNameInvalid        domainerr.Code = "DOMAIN_BENEFIT_NAME_INVALID"
	CodeBenefitUnitInvalid        domainerr.Code = "DOMAIN_BENEFIT_UNIT_INVALID"
	CodeBenefitValueTypeInvalid   domainerr.Code = "DOMAIN_BENEFIT_VALUE_TYPE_INVALID"
	CodeBenefitAggregationInvalid domainerr.Code = "DOMAIN_BENEFIT_AGGREGATION_INVALID"
	CodeBenefitMeteringInvalid    domainerr.Code = "DOMAIN_BENEFIT_METERING_INVALID"
	CodeBenefitUsageResetInvalid  domainerr.Code = "DOMAIN_BENEFIT_USAGE_RESET_INVALID"
	CodeBenefitValueInvalid       domainerr.Code = "DOMAIN_BENEFIT_VALUE_INVALID"
	CodeBenefitPersistenceFailed  domainerr.Code = "DOMAIN_BENEFIT_PERSISTENCE_FAILED"
	CodeBenefitQueryFailed        domainerr.Code = "DOMAIN_BENEFIT_QUERY_FAILED"
)

const (
	CodeMaxLength = 100
	NameMaxLength = 255
	UnitMaxLength = 50
)
