package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"platform-service/internal/app/port"
	"platform-service/internal/interfaces/http/middleware"
)

type fakeOrigins map[string]bool

func (f fakeOrigins) IsAllowedOrigin(_ context.Context, origin string) bool { return f[origin] }

func corsPreflight(static []string, dynamic port.OriginAllowlist, origin string) *httptest.ResponseRecorder {
	r := gin.New()
	r.Use(middleware.CORS(static, dynamic))
	r.POST("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodOptions, "/x", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCORS(t *testing.T) {
	dynamic := fakeOrigins{"https://kasir.example.com": true}
	tests := []struct {
		name        string
		static      []string
		dynamic     port.OriginAllowlist
		origin      string
		wantCode    int
		wantAllowed string
	}{
		{"wildcard", []string{"*"}, dynamic, "https://any.example.com", 204, "*"},
		{"tanpa config & dynamic = semua", nil, nil, "https://any.example.com", 204, "*"},
		{"static", []string{"https://console.example.com"}, dynamic, "https://console.example.com", 204, "https://console.example.com"},
		{"dynamic", []string{"https://console.example.com"}, dynamic, "https://kasir.example.com", 204, "https://kasir.example.com"},
		{"dynamic tanpa static", nil, dynamic, "https://kasir.example.com", 204, "https://kasir.example.com"},
		{"origin asing", []string{"https://console.example.com"}, dynamic, "https://evil.example.com", 403, ""},
		{"origin asing tanpa static", nil, dynamic, "https://evil.example.com", 403, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := corsPreflight(tt.static, tt.dynamic, tt.origin)
			assert.Equal(t, tt.wantCode, w.Code)
			assert.Equal(t, tt.wantAllowed, w.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}
