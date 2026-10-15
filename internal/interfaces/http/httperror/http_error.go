package httperror

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	appErr "platform-service/internal/app/apperror"
	i18n "platform-service/internal/infrastructure/i18n"
	"platform-service/internal/interfaces/http/respond"
)

// globalBundle di-init sekali saat startup via InitBundle().
var globalBundle *i18n.Bundle

// InitBundle menyimpan bundle i18n untuk dipakai di semua error handler.
func InitBundle(b *i18n.Bundle) {
	globalBundle = b
}

type httpError struct {
	statusCode int
	errorCode  string
	message    string
	details    any
}

func (e *httpError) Error() string {
	return fmt.Sprintf("status: %d, code: %s, message: %s", e.statusCode, e.errorCode, e.message)
}

func badRequest(message string) *httpError {
	return &httpError{statusCode: http.StatusBadRequest, errorCode: string(appErr.CodeBadRequest), message: message}
}

func internalError(message string) *httpError {
	return &httpError{statusCode: http.StatusInternalServerError, errorCode: string(appErr.CodeInternal), message: message}
}

// Handle menulis response error yang sesuai untuk err lalu meng-abort request.
// Semua handler dan middleware wajib memakai ini, bukan menulis error sendiri.
func Handle(c *gin.Context, err error) {
	if err == nil {
		return
	}

	locale := c.GetString("locale")
	if locale == "" {
		locale = i18n.DefaultLocale()
	}

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		c.Set("request_err", err)
		c.Set("request_err_code", string(appErr.CodeUnprocessable))
		c.Set("request_err_message", string(appErr.CodeUnprocessable))
		respond.Error(c, http.StatusUnprocessableEntity, string(appErr.CodeUnprocessable), ParseValidationErrors(locale, validationErrors))
		c.Abort()
		return
	}

	httpErr := mapError(err)
	c.Set("request_err", err)
	c.Set("request_err_code", httpErr.errorCode)
	c.Set("request_err_message", httpErr.message)

	payload := any(translate(locale, httpErr.message))
	if httpErr.details != nil {
		payload = httpErr.details
	}
	respond.Error(c, httpErr.statusCode, httpErr.errorCode, payload)
	c.Abort()
}

// translate mengembalikan terjemahan message key. Fallback ke key itu sendiri.
func translate(locale, key string) string {
	if globalBundle != nil {
		return globalBundle.Translate(locale, key)
	}
	return key
}

func mapError(err error) *httpError {
	var appError *appErr.AppError
	if errors.As(err, &appError) {
		return mapAppError(appError)
	}

	if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "EOF") {
		return badRequest("request body is empty or not provided")
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return badRequest(fmt.Sprintf("invalid type for field '%s'", typeErr.Field))
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return badRequest("invalid JSON format")
	}

	// Query/form/uri yang tidak bisa di-parse ke tipe field (mis. limit=abc, since=kemarin).
	var numErr *strconv.NumError
	if errors.As(err, &numErr) {
		return badRequest("invalid number in request parameters")
	}
	var timeErr *time.ParseError
	if errors.As(err, &timeErr) {
		return badRequest("invalid time in request parameters; use RFC3339")
	}

	return internalError(string(appErr.CodeInternal))
}

var statusByCode = map[appErr.Code]int{
	appErr.CodeBadRequest:          http.StatusBadRequest,
	appErr.CodeUnprocessable:       http.StatusUnprocessableEntity,
	appErr.CodeUnauthorized:        http.StatusUnauthorized,
	appErr.CodeForbidden:           http.StatusForbidden,
	appErr.CodeLicenseManaged:      http.StatusForbidden,
	appErr.CodeNotFound:            http.StatusNotFound,
	appErr.CodeRouteNotFound:       http.StatusNotFound,
	appErr.CodeConflict:            http.StatusConflict,
	appErr.CodeServiceUnavailable:  http.StatusServiceUnavailable,
	appErr.CodeUpstreamUnavailable: http.StatusBadGateway,
}

func mapAppError(err *appErr.AppError) *httpError {
	status, ok := statusByCode[err.Code]
	if !ok {
		return internalError(string(appErr.CodeInternal))
	}
	return &httpError{statusCode: status, errorCode: string(err.Code), message: err.Message, details: err.Details}
}
