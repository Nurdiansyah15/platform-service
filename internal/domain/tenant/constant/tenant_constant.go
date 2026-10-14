package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodeTenantNotFound          domainerr.Code = "DOMAIN_TENANT_NOT_FOUND"
	CodeTenantSlugDuplicate     domainerr.Code = "DOMAIN_TENANT_SLUG_DUPLICATE"
	CodeTenantSlugInvalid       domainerr.Code = "DOMAIN_TENANT_SLUG_INVALID"
	CodeTenantNameInvalid       domainerr.Code = "DOMAIN_TENANT_NAME_INVALID"
	CodeTenantStatusInvalid     domainerr.Code = "DOMAIN_TENANT_STATUS_INVALID"
	CodeTenantAccessDenied      domainerr.Code = "DOMAIN_TENANT_ACCESS_DENIED"
	CodeTenantNotSelected       domainerr.Code = "DOMAIN_TENANT_NOT_SELECTED"
	CodeTenantPersistenceFailed domainerr.Code = "DOMAIN_TENANT_PERSISTENCE_FAILED"
	CodeTenantQueryFailed       domainerr.Code = "DOMAIN_TENANT_QUERY_FAILED"

	CodeMemberNotFound         domainerr.Code = "DOMAIN_TENANT_MEMBER_NOT_FOUND"
	CodeMemberAlreadyExists    domainerr.Code = "DOMAIN_TENANT_MEMBER_ALREADY_EXISTS"
	CodeMemberStatusInvalid    domainerr.Code = "DOMAIN_TENANT_MEMBER_STATUS_INVALID"
	CodeMemberUserNotFound     domainerr.Code = "DOMAIN_TENANT_MEMBER_USER_NOT_FOUND"
	CodeMemberIdentityRequired domainerr.Code = "DOMAIN_TENANT_MEMBER_IDENTITY_REQUIRED"
)

const (
	NameMaxLength = 255
	SlugMaxLength = 100
)
