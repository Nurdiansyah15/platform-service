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

	// Permission group & scope type custom (didefinisikan solution lewat sync katalog).
	CodeGroupCodeInvalid       domainerr.Code = "DOMAIN_PERMISSION_GROUP_CODE_INVALID"
	CodeGroupNameInvalid       domainerr.Code = "DOMAIN_PERMISSION_GROUP_NAME_INVALID"
	CodeGroupDuplicate         domainerr.Code = "DOMAIN_PERMISSION_GROUP_DUPLICATE"
	CodeGroupUnknown           domainerr.Code = "DOMAIN_PERMISSION_GROUP_UNKNOWN"
	CodeScopeTypeCodeInvalid   domainerr.Code = "DOMAIN_PERMISSION_SCOPE_TYPE_CODE_INVALID"
	CodeScopeTypeReserved      domainerr.Code = "DOMAIN_PERMISSION_SCOPE_TYPE_RESERVED"
	CodeScopeTypeNameInvalid   domainerr.Code = "DOMAIN_PERMISSION_SCOPE_TYPE_NAME_INVALID"
	CodeScopeTypeParentInvalid domainerr.Code = "DOMAIN_PERMISSION_SCOPE_TYPE_PARENT_INVALID"
	CodeScopeTypeDuplicate     domainerr.Code = "DOMAIN_PERMISSION_SCOPE_TYPE_DUPLICATE"
	CodeScopeTypeUnknown       domainerr.Code = "DOMAIN_PERMISSION_SCOPE_TYPE_UNKNOWN"
	CodeScopeTypeNotFound      domainerr.Code = "DOMAIN_PERMISSION_SCOPE_TYPE_NOT_FOUND"

	// Permission manual (dibuat platform admin) vs permission hasil sync.
	CodePermissionCodeTaken      domainerr.Code = "DOMAIN_PERMISSION_CODE_TAKEN"
	CodePermissionSourceConflict domainerr.Code = "DOMAIN_PERMISSION_SOURCE_CONFLICT"
	CodePermissionSyncManaged    domainerr.Code = "DOMAIN_PERMISSION_SYNC_MANAGED"
	CodePermissionStatusInvalid  domainerr.Code = "DOMAIN_PERMISSION_STATUS_INVALID"

	// Group manual (dibuat platform admin) vs group hasil sync.
	CodeGroupNotFound       domainerr.Code = "DOMAIN_PERMISSION_GROUP_NOT_FOUND"
	CodeGroupCodeTaken      domainerr.Code = "DOMAIN_PERMISSION_GROUP_CODE_TAKEN"
	CodeGroupSourceConflict domainerr.Code = "DOMAIN_PERMISSION_GROUP_SOURCE_CONFLICT"
	CodeGroupSyncManaged    domainerr.Code = "DOMAIN_PERMISSION_GROUP_SYNC_MANAGED"

	// Scope type manual (dibuat platform admin) vs scope type hasil sync.
	CodeScopeTypeCodeTaken      domainerr.Code = "DOMAIN_PERMISSION_SCOPE_TYPE_CODE_TAKEN"
	CodeScopeTypeSourceConflict domainerr.Code = "DOMAIN_PERMISSION_SCOPE_TYPE_SOURCE_CONFLICT"
	CodeScopeTypeSyncManaged    domainerr.Code = "DOMAIN_PERMISSION_SCOPE_TYPE_SYNC_MANAGED"
)

const (
	CodeMaxLength           = 150
	NameMaxLength           = 255
	CatalogVersionMaxLength = 50
	GroupCodeMaxLength      = 100
	ScopeTypeCodeMaxLength  = 30
)

// Scope type built-in yang selalu tersedia di setiap solution (tidak disimpan di tabel).
const (
	ScopeTypeCompany = "company"
	ScopeTypeBranch  = "branch"
)
