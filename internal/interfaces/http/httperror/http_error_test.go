package httperror_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/app/apperror"
	"platform-service/internal/interfaces/http/httperror"
)

type errorBody struct {
	Status     string          `json:"status"`
	StatusCode int             `json:"status_code"`
	ErrorCode  string          `json:"error_code"`
	Errors     json.RawMessage `json:"errors"`
}

func run(t *testing.T, h gin.HandlerFunc, body string) (int, errorBody) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	httperror.RegisterValidatorTagNames()

	r := gin.New()
	r.Use(httperror.Middleware(slogDiscard()))
	r.POST("/x", h)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var out errorBody
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out), w.Body.String())
	return w.Code, out
}

func TestHandle_AppErrorStatusMapping(t *testing.T) {
	tests := []struct {
		err      error
		wantCode int
		wantErr  string
	}{
		{apperror.BadRequest("X"), 400, "ERR_BAD_REQUEST"},
		{apperror.Unauthorized("X"), 401, "ERR_UNAUTHORIZED"},
		{apperror.Forbidden("X"), 403, "ERR_FORBIDDEN"},
		{apperror.LicenseManaged(), 403, "LICENSE_MANAGED"},
		{apperror.NotFound("X"), 404, "ERR_NOT_FOUND"},
		{apperror.Conflict("X"), 409, "ERR_CONFLICT"},
		{apperror.Unprocessable("X", nil), 422, "ERR_UNPROCESSABLE_ENTITY"},
		{apperror.ServiceUnavailable("X"), 503, "ERR_SERVICE_UNAVAILABLE"},
		{apperror.Internal("X", errors.New("db")), 500, "ERR_INTERNAL"},
		{errors.New("error tak dikenal"), 500, "ERR_INTERNAL"},
	}
	for _, tt := range tests {
		t.Run(tt.wantErr, func(t *testing.T) {
			code, body := run(t, func(c *gin.Context) { httperror.Handle(c, tt.err) }, "")
			assert.Equal(t, tt.wantCode, code)
			assert.Equal(t, tt.wantCode, body.StatusCode)
			assert.Equal(t, "error", body.Status)
			assert.Equal(t, tt.wantErr, body.ErrorCode)
		})
	}
}

func TestHandle_InternalErrorDoesNotLeakCause(t *testing.T) {
	_, body := run(t, func(c *gin.Context) {
		httperror.Handle(c, errors.New("pq: password authentication failed for user admin"))
	}, "")
	assert.NotContains(t, string(body.Errors), "password")
}

type createReq struct {
	Name     string `json:"name" binding:"required"`
	TenantID string `json:"tenant_id" binding:"required,uuid"`
}

func TestHandle_ValidationErrorUsesJSONFieldNames(t *testing.T) {
	code, body := run(t, func(c *gin.Context) {
		var req createReq
		if err := c.ShouldBindJSON(&req); err != nil {
			httperror.Handle(c, err)
			return
		}
		c.Status(201)
	}, `{"tenant_id":"bukan-uuid"}`)

	assert.Equal(t, 422, code)
	assert.Equal(t, "ERR_UNPROCESSABLE_ENTITY", body.ErrorCode)
	var fields map[string]string
	require.NoError(t, json.Unmarshal(body.Errors, &fields))
	assert.Contains(t, fields, "name")
	assert.Contains(t, fields, "tenant_id")
}

func TestHandle_MalformedJSON(t *testing.T) {
	code, body := run(t, func(c *gin.Context) {
		var req createReq
		httperror.Handle(c, c.ShouldBindJSON(&req))
	}, `{"name":`)
	assert.Equal(t, 400, code)
	assert.Equal(t, "ERR_BAD_REQUEST", body.ErrorCode)
}

func TestHandle_UnparsableQueryParams(t *testing.T) {
	type query struct {
		Limit *int       `form:"limit"`
		Since *time.Time `form:"since" time_format:"2006-01-02T15:04:05Z07:00"`
	}
	for _, raw := range []string{"limit=abc", "since=kemarin"} {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/x", func(c *gin.Context) {
			var q query
			httperror.Handle(c, c.ShouldBindQuery(&q))
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x?"+raw, nil))
		assert.Equal(t, 400, w.Code, raw)
	}
}

func TestMiddleware_RecoversPanic(t *testing.T) {
	code, body := run(t, func(*gin.Context) { panic("boom") }, "")
	assert.Equal(t, 500, code)
	assert.Equal(t, "ERR_INTERNAL", body.ErrorCode)
}
