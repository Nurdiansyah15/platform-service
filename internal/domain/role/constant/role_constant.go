package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodeRoleNotFound             domainerr.Code = "DOMAIN_ROLE_NOT_FOUND"
	CodeRoleCodeDuplicate        domainerr.Code = "DOMAIN_ROLE_CODE_DUPLICATE"
	CodeRoleCodeInvalid          domainerr.Code = "DOMAIN_ROLE_CODE_INVALID"
	CodeRoleNameInvalid          domainerr.Code = "DOMAIN_ROLE_NAME_INVALID"
	CodeRoleSystemImmutable      domainerr.Code = "DOMAIN_ROLE_SYSTEM_IMMUTABLE"
	CodeRoleTenantNotSubscribed  domainerr.Code = "DOMAIN_ROLE_TENANT_NOT_SUBSCRIBED"
	CodeRolePermissionUnknown    domainerr.Code = "DOMAIN_ROLE_PERMISSION_UNKNOWN"
	CodeRolePermissionDuplicate  domainerr.Code = "DOMAIN_ROLE_PERMISSION_DUPLICATE"
	CodeRolePermissionDeprecated domainerr.Code = "DOMAIN_ROLE_PERMISSION_DEPRECATED"
	CodeRolePersistenceFailed    domainerr.Code = "DOMAIN_ROLE_PERSISTENCE_FAILED"
	CodeRoleQueryFailed          domainerr.Code = "DOMAIN_ROLE_QUERY_FAILED"
)

const (
	CodeMaxLength = 100
	NameMaxLength = 255
)
