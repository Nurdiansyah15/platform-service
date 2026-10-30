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
	CodeInvalidCredentials   domainerr.Code = "DOMAIN_AUTH_INVALID_CREDENTIALS"
	CodeUserNotActive        domainerr.Code = "DOMAIN_AUTH_USER_NOT_ACTIVE"
	CodeSolutionUnavailable  domainerr.Code = "DOMAIN_AUTH_SOLUTION_UNAVAILABLE"
	CodeRefreshTokenInvalid  domainerr.Code = "DOMAIN_AUTH_REFRESH_TOKEN_INVALID"
	CodeCompanyNotSelectable domainerr.Code = "DOMAIN_AUTH_COMPANY_NOT_SELECTABLE"

	// Error alur SSO (authorization code + PKCE).
	// CodeAuthorizeConsoleOnly — authorize hanya boleh dipanggil dengan token sesi console.
	CodeAuthorizeConsoleOnly domainerr.Code = "DOMAIN_AUTH_AUTHORIZE_CONSOLE_ONLY"
	CodeInvalidRedirectURI   domainerr.Code = "DOMAIN_AUTH_INVALID_REDIRECT_URI"
	// CodeInvalidGrant — satu kode untuk semua kegagalan tukar kode (tidak ada, dipakai,
	// kedaluwarsa, PKCE / redirect_uri tidak cocok) agar penyebabnya tidak bocor.
	CodeInvalidGrant              domainerr.Code = "DOMAIN_AUTH_INVALID_GRANT"
	CodeAuthCodeNotFound          domainerr.Code = "DOMAIN_AUTH_CODE_NOT_FOUND"
	CodeAuthCodeAlreadyUsed       domainerr.Code = "DOMAIN_AUTH_CODE_ALREADY_USED"
	CodeAuthCodePersistenceFailed domainerr.Code = "DOMAIN_AUTH_CODE_PERSISTENCE_FAILED"
	CodeAuthCodeQueryFailed       domainerr.Code = "DOMAIN_AUTH_CODE_QUERY_FAILED"
	CodeAuthCodeChallengeInvalid  domainerr.Code = "DOMAIN_AUTH_CODE_CHALLENGE_INVALID"
)

const (
	// CodeChallengeMethodS256 — satu-satunya metode PKCE yang didukung (plain ditolak).
	CodeChallengeMethodS256 = "S256"
)
