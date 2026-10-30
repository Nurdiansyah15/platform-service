package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodeSolutionNotFound      domainerr.Code = "DOMAIN_SOLUTION_NOT_FOUND"
	CodeSolutionCodeDuplicate domainerr.Code = "DOMAIN_SOLUTION_CODE_DUPLICATE"
	CodeSolutionCodeInvalid   domainerr.Code = "DOMAIN_SOLUTION_CODE_INVALID"
	CodeSolutionNameInvalid   domainerr.Code = "DOMAIN_SOLUTION_NAME_INVALID"
	CodeSolutionStatusInvalid domainerr.Code = "DOMAIN_SOLUTION_STATUS_INVALID"
	// CodeSolutionURLInvalid — launch_url / icon_url / redirect_uris bukan URL absolut
	// https (atau http untuk localhost), atau jumlah redirect_uris melebihi batas.
	CodeSolutionURLInvalid domainerr.Code = "DOMAIN_SOLUTION_URL_INVALID"
	// CodeSolutionInUseBySession — solution sesi admin sendiri tidak boleh dinonaktifkan (mencegah terkunci).
	CodeSolutionInUseBySession    domainerr.Code = "DOMAIN_SOLUTION_IN_USE_BY_SESSION"
	CodeSolutionPersistenceFailed domainerr.Code = "DOMAIN_SOLUTION_PERSISTENCE_FAILED"
	CodeSolutionQueryFailed       domainerr.Code = "DOMAIN_SOLUTION_QUERY_FAILED"
)

const (
	CodeMaxLength = 100
	NameMaxLength = 255
	URLMaxLength  = 2000
	// RedirectURIsMax — batas jumlah redirect_uri per solution.
	RedirectURIsMax = 10
)
