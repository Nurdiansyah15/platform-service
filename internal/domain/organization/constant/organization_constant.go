package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodeCompanyNotFound          domainerr.Code = "DOMAIN_COMPANY_NOT_FOUND"
	CodeCompanySlugDuplicate     domainerr.Code = "DOMAIN_COMPANY_SLUG_DUPLICATE"
	CodeCompanySlugInvalid       domainerr.Code = "DOMAIN_COMPANY_SLUG_INVALID"
	CodeCompanyNameInvalid       domainerr.Code = "DOMAIN_COMPANY_NAME_INVALID"
	CodeCompanyStatusInvalid     domainerr.Code = "DOMAIN_COMPANY_STATUS_INVALID"
	CodeCompanyAccessDenied      domainerr.Code = "DOMAIN_COMPANY_ACCESS_DENIED"
	CodeCompanyNotSelected       domainerr.Code = "DOMAIN_COMPANY_NOT_SELECTED"
	CodeCompanyPersistenceFailed domainerr.Code = "DOMAIN_COMPANY_PERSISTENCE_FAILED"
	CodeCompanyQueryFailed       domainerr.Code = "DOMAIN_COMPANY_QUERY_FAILED"

	CodeMemberNotFound         domainerr.Code = "DOMAIN_COMPANY_MEMBER_NOT_FOUND"
	CodeMemberAlreadyExists    domainerr.Code = "DOMAIN_COMPANY_MEMBER_ALREADY_EXISTS"
	CodeMemberStatusInvalid    domainerr.Code = "DOMAIN_COMPANY_MEMBER_STATUS_INVALID"
	CodeMemberUserNotFound     domainerr.Code = "DOMAIN_COMPANY_MEMBER_USER_NOT_FOUND"
	CodeMemberIdentityRequired domainerr.Code = "DOMAIN_COMPANY_MEMBER_IDENTITY_REQUIRED"

	CodeBranchNotFound      domainerr.Code = "DOMAIN_BRANCH_NOT_FOUND"
	CodeBranchCodeDuplicate domainerr.Code = "DOMAIN_BRANCH_CODE_DUPLICATE"

	CodeOrgCodeInvalid       domainerr.Code = "DOMAIN_ORG_CODE_INVALID"
	CodeOrgNameInvalid       domainerr.Code = "DOMAIN_ORG_NAME_INVALID"
	CodeOrgStatusInvalid     domainerr.Code = "DOMAIN_ORG_STATUS_INVALID"
	CodeOrgPersistenceFailed domainerr.Code = "DOMAIN_ORG_PERSISTENCE_FAILED"
	CodeOrgQueryFailed       domainerr.Code = "DOMAIN_ORG_QUERY_FAILED"
)

const (
	NameMaxLength = 255
	SlugMaxLength = 100
	CodeMaxLength = 100
)
