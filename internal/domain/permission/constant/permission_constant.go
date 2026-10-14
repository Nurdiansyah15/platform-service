package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodePermissionNotFound          domainerr.Code = "DOMAIN_PERMISSION_NOT_FOUND"
	CodePermissionCodeInvalid       domainerr.Code = "DOMAIN_PERMISSION_CODE_INVALID"
	CodePermissionNameInvalid       domainerr.Code = "DOMAIN_PERMISSION_NAME_INVALID"
	CodePermissionCodeDuplicate     domainerr.Code = "DOMAIN_PERMISSION_CODE_DUPLICATE"
	CodeCatalogVersionInvalid       domainerr.Code = "DOMAIN_PERMISSION_CATALOG_VERSION_INVALID"
	CodeCatalogSyncForbidden        domainerr.Code = "DOMAIN_PERMISSION_CATALOG_SYNC_FORBIDDEN"
	CodePermissionPersistenceFailed domainerr.Code = "DOMAIN_PERMISSION_PERSISTENCE_FAILED"
	CodePermissionQueryFailed       domainerr.Code = "DOMAIN_PERMISSION_QUERY_FAILED"
)

const (
	CodeMaxLength           = 150
	NameMaxLength           = 255
	CatalogVersionMaxLength = 50
)
