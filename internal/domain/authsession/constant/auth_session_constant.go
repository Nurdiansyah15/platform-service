package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodeSessionNotFound          domainerr.Code = "DOMAIN_AUTH_SESSION_NOT_FOUND"
	CodeSessionRevoked           domainerr.Code = "DOMAIN_AUTH_SESSION_REVOKED"
	CodeSessionExpired           domainerr.Code = "DOMAIN_AUTH_SESSION_EXPIRED"
	CodeSessionRotationConflict  domainerr.Code = "DOMAIN_AUTH_SESSION_ROTATION_CONFLICT"
	CodeSessionInvalid           domainerr.Code = "DOMAIN_AUTH_SESSION_INVALID"
	CodeSessionPersistenceFailed domainerr.Code = "DOMAIN_AUTH_SESSION_PERSISTENCE_FAILED"
	CodeSessionQueryFailed       domainerr.Code = "DOMAIN_AUTH_SESSION_QUERY_FAILED"

	// Error alur autentikasi (dipetakan ke 401/403/422 oleh usecase auth).
	CodeInvalidCredentials  domainerr.Code = "DOMAIN_AUTH_INVALID_CREDENTIALS"
	CodeUserNotActive       domainerr.Code = "DOMAIN_AUTH_USER_NOT_ACTIVE"
	CodeSolutionUnavailable domainerr.Code = "DOMAIN_AUTH_SOLUTION_UNAVAILABLE"
	CodeRefreshTokenInvalid domainerr.Code = "DOMAIN_AUTH_REFRESH_TOKEN_INVALID"
)
