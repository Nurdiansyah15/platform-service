package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodeClientNotFound          domainerr.Code = "DOMAIN_SOLUTION_CLIENT_NOT_FOUND"
	CodeClientNameInvalid       domainerr.Code = "DOMAIN_SOLUTION_CLIENT_NAME_INVALID"
	CodeClientScopeInvalid      domainerr.Code = "DOMAIN_SOLUTION_CLIENT_SCOPE_INVALID"
	CodeClientRevoked           domainerr.Code = "DOMAIN_SOLUTION_CLIENT_REVOKED"
	CodeClientInvalid           domainerr.Code = "DOMAIN_SOLUTION_CLIENT_INVALID"
	CodeClientPersistenceFailed domainerr.Code = "DOMAIN_SOLUTION_CLIENT_PERSISTENCE_FAILED"
	CodeClientQueryFailed       domainerr.Code = "DOMAIN_SOLUTION_CLIENT_QUERY_FAILED"
)

const NameMaxLength = 255

// Daftar scope service (spec §2.8).
const (
	ScopeUsersRead        = "users:read"
	ScopeUsersWrite       = "users:write"
	ScopeOrgRead          = "org:read"
	ScopeOrgWrite         = "org:write"
	ScopeAuthzCheck       = "authz:check"
	ScopeRolesWrite       = "roles:write"
	ScopeEntitlementsRead = "entitlements:read"
	ScopeUsageWrite       = "usage:write"
	ScopeCatalogSync      = "catalog:sync"
	ScopeEventsRead       = "events:read"
	ScopeLicenseRead      = "license:read"
)

// AllScopes adalah seluruh scope yang dikenal, urut sesuai spec.
var AllScopes = []string{
	ScopeUsersRead, ScopeUsersWrite, ScopeOrgRead, ScopeOrgWrite, ScopeAuthzCheck, ScopeRolesWrite,
	ScopeEntitlementsRead, ScopeUsageWrite, ScopeCatalogSync, ScopeEventsRead, ScopeLicenseRead,
}
