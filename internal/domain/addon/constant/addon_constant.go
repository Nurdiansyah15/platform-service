package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodeAddonNotFound          domainerr.Code = "DOMAIN_ADDON_NOT_FOUND"
	CodeAddonCodeDuplicate     domainerr.Code = "DOMAIN_ADDON_CODE_DUPLICATE"
	CodeAddonCodeInvalid       domainerr.Code = "DOMAIN_ADDON_CODE_INVALID"
	CodeAddonNameInvalid       domainerr.Code = "DOMAIN_ADDON_NAME_INVALID"
	CodeAddonPersistenceFailed domainerr.Code = "DOMAIN_ADDON_PERSISTENCE_FAILED"
	CodeAddonQueryFailed       domainerr.Code = "DOMAIN_ADDON_QUERY_FAILED"
)

const (
	CodeMaxLength = 100
	NameMaxLength = 255
)
