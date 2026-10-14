package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"platform-service/internal/config"
	"platform-service/internal/interfaces/http/middleware"
)

func serveGuard(mode config.DeploymentMode, pick func(*middleware.DeploymentGuard) gin.HandlerFunc) (int, string) {
	r := gin.New()
	r.POST("/x", pick(middleware.NewDeploymentGuard(mode)), func(c *gin.Context) { c.Status(http.StatusCreated) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/x", nil))

	var body struct {
		ErrorCode string `json:"error_code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body.ErrorCode
}

func TestDeploymentGuard(t *testing.T) {
	licenseManaged := func(g *middleware.DeploymentGuard) gin.HandlerFunc { return g.LicenseManaged() }
	onPremOnly := func(g *middleware.DeploymentGuard) gin.HandlerFunc { return g.OnPremOnly() }
	cloudOnly := func(g *middleware.DeploymentGuard) gin.HandlerFunc { return g.CloudOnly() }

	tests := []struct {
		name      string
		mode      config.DeploymentMode
		pick      func(*middleware.DeploymentGuard) gin.HandlerFunc
		wantCode  int
		wantError string
	}{
		{"LicenseManaged/cloud lolos", config.DeploymentCloud, licenseManaged, 201, ""},
		{"LicenseManaged/on_prem ditolak", config.DeploymentOnPrem, licenseManaged, 403, "LICENSE_MANAGED"},
		{"OnPremOnly/on_prem lolos", config.DeploymentOnPrem, onPremOnly, 201, ""},
		{"OnPremOnly/cloud 404", config.DeploymentCloud, onPremOnly, 404, "ERR_ROUTE_NOT_FOUND"},
		{"CloudOnly/cloud lolos", config.DeploymentCloud, cloudOnly, 201, ""},
		{"CloudOnly/on_prem 404", config.DeploymentOnPrem, cloudOnly, 404, "ERR_ROUTE_NOT_FOUND"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, errCode := serveGuard(tt.mode, tt.pick)
			assert.Equal(t, tt.wantCode, code)
			assert.Equal(t, tt.wantError, errCode)
		})
	}
}
