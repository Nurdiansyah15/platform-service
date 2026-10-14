package api_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/testhelper"
)

// ── Health ───────────────────────────────────────────────────────────────────

func TestSystem_Health_Success(t *testing.T) {
	w := testSrv.GET("/healthz")
	require.Equal(t, 200, w.Code, "health: %s", w.Body.String())

	var resp struct {
		Status     string `json:"status"`
		StatusCode int    `json:"status_code"`
		Data       struct {
			Status         string `json:"status"`
			Database       string `json:"database"`
			DeploymentMode string `json:"deployment_mode"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "ok", resp.Data.Status)
	assert.Equal(t, "ok", resp.Data.Database)
	assert.Equal(t, "cloud", resp.Data.DeploymentMode)
}

func TestSystem_Health_SetsRequestID(t *testing.T) {
	w := testSrv.GET("/healthz", map[string]string{"X-Request-Id": "client-123"})
	require.Equal(t, 200, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-Id"))
	assert.NotEqual(t, "client-123", w.Header().Get("X-Request-Id"), "request id internal tidak boleh dari client")
	assert.Equal(t, "client-123", w.Header().Get("X-Client-Request-Id"))
}

// ── Route tidak terdaftar ────────────────────────────────────────────────────

func TestSystem_UnknownRoute_NotFound(t *testing.T) {
	w := testSrv.GET("/api/v1/route-yang-tidak-ada")
	require.Equal(t, 404, w.Code)

	body := testhelper.DecodeError(t, w)
	assert.Equal(t, "error", body.Status)
	assert.Equal(t, "ERR_ROUTE_NOT_FOUND", body.ErrorCode)
}

func TestSystem_UnknownRoute_TranslatedByLocale(t *testing.T) {
	w := testSrv.GET("/api/v1/route-yang-tidak-ada", map[string]string{"Accept-Language": "id-ID,id;q=0.9"})
	require.Equal(t, 404, w.Code)

	var msg string
	require.NoError(t, json.Unmarshal(testhelper.DecodeError(t, w).Errors, &msg))
	assert.Equal(t, "Rute yang diminta tidak ditemukan.", msg)
}
