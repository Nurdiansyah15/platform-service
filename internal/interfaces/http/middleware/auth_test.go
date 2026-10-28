package middleware_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/app/principal"
	"platform-service/internal/interfaces/http/middleware"
)

const (
	tokenUser          = "user-no-company"
	tokenUserCompany   = "user-with-company"
	tokenPlatformAdmin = "platform-admin"
	tokenServiceUsage  = "service-usage"
)

// fakeAuthenticator memetakan token statis ke principal.
type fakeAuthenticator map[string]*principal.Principal

func (f fakeAuthenticator) Authenticate(_ context.Context, token string) (*principal.Principal, error) {
	if p, ok := f[token]; ok {
		return p, nil
	}
	return nil, errors.New("unknown token")
}

var authn = fakeAuthenticator{
	tokenUser:          {Kind: principal.KindUser, UserID: "u1"},
	tokenUserCompany:   {Kind: principal.KindUser, UserID: "u2", CompanyID: "t1"},
	tokenPlatformAdmin: {Kind: principal.KindUser, UserID: "u3", IsPlatformAdmin: true},
	tokenServiceUsage:  {Kind: principal.KindService, ClientID: "c1", SolutionID: "s1", Scopes: []string{"usage:write"}},
}

func init() {
	gin.SetMode(gin.TestMode)
}

// serve menjalankan satu request ke route yang dilindungi Authenticate + guards.
func serve(t *testing.T, token string, guards ...gin.HandlerFunc) (int, string) {
	t.Helper()
	r := gin.New()
	handlers := append([]gin.HandlerFunc{middleware.Authenticate(authn)}, guards...)
	handlers = append(handlers, func(c *gin.Context) {
		p, ok := middleware.PrincipalFrom(c)
		require.True(t, ok)
		c.JSON(http.StatusOK, gin.H{"kind": p.Kind})
	})
	r.GET("/protected", handlers...)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body struct {
		ErrorCode string `json:"error_code"`
		Errors    any    `json:"errors"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	msg, _ := body.Errors.(string)
	return w.Code, msg
}

// ── Authenticate ─────────────────────────────────────────────────────────────

func TestAuthenticate_MissingToken(t *testing.T) {
	code, msg := serve(t, "")
	assert.Equal(t, 401, code)
	assert.Equal(t, "AUTH_TOKEN_MISSING", msg)
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	code, msg := serve(t, "garbage")
	assert.Equal(t, 401, code)
	assert.Equal(t, "AUTH_TOKEN_INVALID", msg)
}

func TestAuthenticate_NonBearerScheme(t *testing.T) {
	r := gin.New()
	r.GET("/protected", middleware.Authenticate(authn), func(c *gin.Context) { c.Status(200) })
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic "+tokenUser)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 401, w.Code)
}

func TestAuthenticate_Success(t *testing.T) {
	code, _ := serve(t, tokenUser)
	assert.Equal(t, 200, code)
}

// ── Guards ───────────────────────────────────────────────────────────────────

func TestGuards(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		guard    gin.HandlerFunc
		wantCode int
		wantMsg  string
	}{
		{"RequireUser/user", tokenUser, middleware.RequireUser(), 200, ""},
		{"RequireUser/service", tokenServiceUsage, middleware.RequireUser(), 403, "AUTH_USER_TOKEN_REQUIRED"},

		{"RequireService/scope ada", tokenServiceUsage, middleware.RequireService("usage:write"), 200, ""},
		{"RequireService/scope kurang", tokenServiceUsage, middleware.RequireService("catalog:sync"), 403, "AUTH_INSUFFICIENT_SCOPE"},
		{"RequireService/user", tokenUser, middleware.RequireService("usage:write"), 403, "AUTH_SERVICE_TOKEN_REQUIRED"},

		{"RequireUserOrService/user", tokenUser, middleware.RequireUserOrService("entitlements:read"), 200, ""},
		{"RequireUserOrService/service scope ada", tokenServiceUsage, middleware.RequireUserOrService("usage:write"), 200, ""},
		{"RequireUserOrService/service scope kurang", tokenServiceUsage, middleware.RequireUserOrService("entitlements:read"), 403, "AUTH_INSUFFICIENT_SCOPE"},

		{"RequirePlatformAdmin/admin", tokenPlatformAdmin, middleware.RequirePlatformAdmin(), 200, ""},
		{"RequirePlatformAdmin/user biasa", tokenUser, middleware.RequirePlatformAdmin(), 403, "AUTH_PLATFORM_ADMIN_REQUIRED"},
		{"RequirePlatformAdmin/service", tokenServiceUsage, middleware.RequirePlatformAdmin(), 403, "AUTH_PLATFORM_ADMIN_REQUIRED"},

		{"RequireActiveCompany/company terpilih", tokenUserCompany, middleware.RequireActiveCompany(), 200, ""},
		{"RequireActiveCompany/cid kosong", tokenUser, middleware.RequireActiveCompany(), 403, "AUTH_COMPANY_NOT_SELECTED"},
		{"RequireActiveCompany/service", tokenServiceUsage, middleware.RequireActiveCompany(), 200, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, msg := serve(t, tt.token, tt.guard)
			assert.Equal(t, tt.wantCode, code)
			if tt.wantMsg != "" {
				assert.Equal(t, tt.wantMsg, msg)
			}
		})
	}
}

func TestGuard_WithoutAuthenticate_Unauthorized(t *testing.T) {
	r := gin.New()
	r.GET("/x", middleware.RequireUser(), func(c *gin.Context) { c.Status(200) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	assert.Equal(t, 401, w.Code)
}
