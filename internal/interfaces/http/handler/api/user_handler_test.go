package api_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/testhelper"
)

type userData struct {
	ID              string `json:"id"`
	Email           string `json:"email"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	IsPlatformAdmin bool   `json:"is_platform_admin"`
	HasPassword     bool   `json:"has_password"`
}

// ── Create ───────────────────────────────────────────────────────────────────

func TestUser_Create_Unauthenticated(t *testing.T) {
	w := testSrv.POST("/api/v1/users", map[string]any{"email": testhelper.UniqueEmail("u"), "name": "X"})
	assert.Equal(t, 401, w.Code)
}

func TestUser_Create_Forbidden_NonAdmin(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)
	w := testSrv.POST("/api/v1/users", map[string]any{"email": testhelper.UniqueEmail("u"), "name": "X"},
		testhelper.BearerHeader(tok.AccessToken))
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestUser_Create_Forbidden_ServiceToken(t *testing.T) {
	w := testSrv.POST("/api/v1/users", map[string]any{"email": testhelper.UniqueEmail("u"), "name": "X"},
		testhelper.BearerHeader(testSrv.Auth.ServiceToken(uuid.NewString(), "users:write")))
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestUser_Create_InvalidPayload(t *testing.T) {
	tests := map[string]map[string]any{
		"tanpa email":        {"name": "X"},
		"password pendek":    {"email": testhelper.UniqueEmail("u"), "name": "X", "password": "short"},
		"status tak dikenal": {"email": testhelper.UniqueEmail("u"), "name": "X", "status": "banned"},
		"email tidak valid":  {"email": "bukan-email", "name": "X"},
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			w := testSrv.POST("/api/v1/users", body, adminHeader())
			assert.Equal(t, 422, w.Code, w.Body.String())
		})
	}
}

func TestUser_Create_DuplicateEmailCaseInsensitive(t *testing.T) {
	email := testhelper.UniqueEmail("dup")
	mustCreateUser(t, email)

	w := testSrv.POST("/api/v1/users", map[string]any{"email": upper(email), "name": "Dup"}, adminHeader())
	require.Equal(t, 409, w.Code, w.Body.String())
	assert.Equal(t, "ERR_CONFLICT", testhelper.DecodeError(t, w).ErrorCode)
}

func TestUser_Create_Success(t *testing.T) {
	email := testhelper.UniqueEmail("New.User")
	w := testSrv.POST("/api/v1/users", map[string]any{
		"email":    email,
		"name":     "  Budi  ",
		"password": testPassword,
	}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())

	var got userData
	testhelper.DecodeBody(t, w, &got)
	assert.NotEmpty(t, got.ID)
	assert.Equal(t, lower(email), got.Email)
	assert.Equal(t, "Budi", got.Name)
	assert.Equal(t, "active", got.Status)
	assert.False(t, got.IsPlatformAdmin)
	assert.True(t, got.HasPassword)
	assert.NotContains(t, w.Body.String(), "password")

	var hash string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT password_hash FROM users WHERE id = $1`, got.ID).Scan(&hash))
	assert.NotEqual(t, testPassword, hash)

	// User baru bisa langsung login.
	mustLogin(t, email, testConsoleSolution)
}

func TestUser_Create_PlatformAdmin(t *testing.T) {
	email := testhelper.UniqueEmail("admin2")
	w := testSrv.POST("/api/v1/users", map[string]any{
		"email": email, "name": "Admin 2", "password": testPassword, "is_platform_admin": true,
	}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())

	tok := mustLogin(t, email, testConsoleSolution)
	w = testSrv.POST("/api/v1/solutions", map[string]any{"code": testhelper.UniqueCode("byadmin2"), "name": "X"},
		testhelper.BearerHeader(tok.AccessToken))
	assert.Equal(t, 201, w.Code, "admin baru langsung punya akses admin: %s", w.Body.String())
}
