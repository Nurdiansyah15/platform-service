package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodeCompanyNotFound      domainerr.Code = "DOMAIN_COMPANY_NOT_FOUND"
	CodeCompanyCodeDuplicate domainerr.Code = "DOMAIN_COMPANY_CODE_DUPLICATE"
	CodeBranchNotFound       domainerr.Code = "DOMAIN_BRANCH_NOT_FOUND"
	CodeBranchCodeDuplicate  domainerr.Code = "DOMAIN_BRANCH_CODE_DUPLICATE"

	CodeOrgCodeInvalid       domainerr.Code = "DOMAIN_ORG_CODE_INVALID"
	CodeOrgNameInvalid       domainerr.Code = "DOMAIN_ORG_NAME_INVALID"
	CodeOrgStatusInvalid     domainerr.Code = "DOMAIN_ORG_STATUS_INVALID"
	CodeOrgPersistenceFailed domainerr.Code = "DOMAIN_ORG_PERSISTENCE_FAILED"
	CodeOrgQueryFailed       domainerr.Code = "DOMAIN_ORG_QUERY_FAILED"
)

const (
	NameMaxLength = 255
	CodeMaxLength = 100
)
