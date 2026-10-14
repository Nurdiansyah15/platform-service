package apperror

import "fmt"

type Code string

const (
	CodeBadRequest          Code = "ERR_BAD_REQUEST"
	CodeUnprocessable       Code = "ERR_UNPROCESSABLE_ENTITY"
	CodeUnauthorized        Code = "ERR_UNAUTHORIZED"
	CodeForbidden           Code = "ERR_FORBIDDEN"
	CodeNotFound            Code = "ERR_NOT_FOUND"
	CodeRouteNotFound       Code = "ERR_ROUTE_NOT_FOUND"
	CodeConflict            Code = "ERR_CONFLICT"
	CodeInternal            Code = "ERR_INTERNAL"
	CodeServiceUnavailable  Code = "ERR_SERVICE_UNAVAILABLE"
	CodeUpstreamUnavailable Code = "ERR_UPSTREAM_UNAVAILABLE"

	// CodeLicenseManaged — endpoint dinonaktifkan di mode on_prem karena datanya
	// dikelola lewat license (spec §2.14). Dipetakan ke 403.
	CodeLicenseManaged Code = "LICENSE_MANAGED"
)

// Message key lintas domain (key terjemahan di locales/*.json).
const (
	MsgTokenMissing          = "AUTH_TOKEN_MISSING"
	MsgTokenInvalid          = "AUTH_TOKEN_INVALID"
	MsgUserTokenRequired     = "AUTH_USER_TOKEN_REQUIRED"
	MsgServiceTokenRequired  = "AUTH_SERVICE_TOKEN_REQUIRED"
	MsgInsufficientScope     = "AUTH_INSUFFICIENT_SCOPE"
	MsgPlatformAdminRequired = "AUTH_PLATFORM_ADMIN_REQUIRED"
	MsgTenantNotSelected     = "AUTH_TENANT_NOT_SELECTED"
)

type AppError struct {
	Code    Code
	Message string
	Details any
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("code: %s, message: %s, err: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("code: %s, message: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code Code, message string, details any) *AppError {
	return &AppError{Code: code, Message: message, Details: details}
}

func Wrap(code Code, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

func BadRequest(message string, cause ...error) *AppError {
	return withCause(New(CodeBadRequest, message, nil), cause)
}

// Unprocessable dipakai untuk validasi bisnis; details (mis. map field → pesan)
// dikirim apa adanya di response.
func Unprocessable(message string, details any, cause ...error) *AppError {
	return withCause(New(CodeUnprocessable, message, details), cause)
}

func Unauthorized(message string, cause ...error) *AppError {
	return withCause(New(CodeUnauthorized, message, nil), cause)
}

func Forbidden(message string, cause ...error) *AppError {
	return withCause(New(CodeForbidden, message, nil), cause)
}

func NotFound(message string, cause ...error) *AppError {
	return withCause(New(CodeNotFound, message, nil), cause)
}

func RouteNotFound() *AppError {
	return New(CodeRouteNotFound, string(CodeRouteNotFound), nil)
}

func Conflict(message string, cause ...error) *AppError {
	return withCause(New(CodeConflict, message, nil), cause)
}

func LicenseManaged() *AppError {
	return New(CodeLicenseManaged, string(CodeLicenseManaged), nil)
}

func ServiceUnavailable(message string, cause ...error) *AppError {
	return withCause(New(CodeServiceUnavailable, message, nil), cause)
}

func UpstreamUnavailable(message string, err error) *AppError {
	return Wrap(CodeUpstreamUnavailable, message, err)
}

func Internal(message string, err error) *AppError {
	return Wrap(CodeInternal, message, err)
}

func withCause(ae *AppError, causes []error) *AppError {
	for _, err := range causes {
		if err != nil {
			ae.Err = err
			break
		}
	}
	return ae
}
