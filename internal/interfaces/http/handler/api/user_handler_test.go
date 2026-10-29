package api_test

import (
	"context"
	"encoding/json"
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

// ── Helpers ──────────────────────────────────────────────────────────────────

func userPath(id string) string {
	return "/api/v1/users/" + id
}

// mustLoginNewAdmin membuat platform admin baru lalu login; mengembalikan header & ID-nya.
func mustLoginNewAdmin(t *testing.T) (map[string]string, string) {
	t.Helper()
	email := testhelper.UniqueEmail("admin")
	w := testSrv.POST("/api/v1/users", map[string]any{
		"email": email, "name": "Admin", "password": testPassword, "is_platform_admin": true,
	}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())
	var u userData
	testhelper.DecodeBody(t, w, &u)
	code := testhelper.UniqueCode("adm")
	testhelper.MustSeedSolution(t, testSrv.DB, code, "active")
	return testhelper.BearerHeader(mustLogin(t, email, code).AccessToken), u.ID
}

func countUserEvents(t *testing.T, eventType, userID string) int {
	t.Helper()
	var n int
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM events WHERE event_type = $1 AND payload->>'user_id' = $2`, eventType, userID).Scan(&n))
	return n
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestUser_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET("/api/v1/users").Code)
}

func TestUser_List_Forbidden_NonAdmin(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)
	assert.Equal(t, 403, testSrv.GET("/api/v1/users", testhelper.BearerHeader(tok.AccessToken)).Code)
}

func TestUser_List_InvalidQuery(t *testing.T) {
	assert.Equal(t, 422, testSrv.GET("/api/v1/users?status=deleted", adminHeader()).Code)
}

func TestUser_List_Success(t *testing.T) {
	marker := testhelper.UniqueCode("lst")
	active := mustCreateUser(t, marker+"-a@example.com")
	suspended := mustCreateUser(t, marker+"-b@example.com")
	require.Equal(t, 200, testSrv.PATCH(userPath(suspended)+"/status", map[string]any{"status": "suspended"}, adminHeader()).Code)

	var resp struct {
		Data []userData `json:"data"`
		Meta struct {
			Total int64 `json:"total"`
		} `json:"meta"`
	}
	w := testSrv.GET("/api/v1/users?search="+marker, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, int64(2), resp.Meta.Total)

	w = testSrv.GET("/api/v1/users?status=active&search="+marker, adminHeader())
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, active, resp.Data[0].ID)
	assert.True(t, resp.Data[0].HasPassword)
}

// ── Get ──────────────────────────────────────────────────────────────────────

func TestUser_Get_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(userPath(uuid.NewString())).Code)
}

func TestUser_Get_Forbidden_NonAdmin(t *testing.T) {
	tok, userID, _ := mustLoginNewUser(t)
	assert.Equal(t, 403, testSrv.GET(userPath(userID), testhelper.BearerHeader(tok.AccessToken)).Code)
}

func TestUser_Get_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.GET(userPath(uuid.NewString()), adminHeader()).Code)
	assert.Equal(t, 422, testSrv.GET(userPath("bukan-uuid"), adminHeader()).Code)
}

func TestUser_Get_Success_WithMemberships(t *testing.T) {
	_, ownerID, companyID := mustCompanyOwner(t)

	w := testSrv.GET(userPath(ownerID), adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got struct {
		ID          string `json:"id"`
		Memberships []struct {
			CompanyID    string `json:"company_id"`
			MemberStatus string `json:"member_status"`
		} `json:"memberships"`
	}
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, ownerID, got.ID)
	require.Len(t, got.Memberships, 1)
	assert.Equal(t, companyID, got.Memberships[0].CompanyID)
	assert.Equal(t, "active", got.Memberships[0].MemberStatus)
}

// ── Update ───────────────────────────────────────────────────────────────────

func TestUser_Update_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.PATCH(userPath(uuid.NewString()), map[string]any{"name": "X"}).Code)
}

func TestUser_Update_Forbidden_NonAdmin(t *testing.T) {
	tok, userID, _ := mustLoginNewUser(t)
	assert.Equal(t, 403, testSrv.PATCH(userPath(userID), map[string]any{"name": "X"}, testhelper.BearerHeader(tok.AccessToken)).Code)
}

func TestUser_Update_Forbidden_RemoveOwnAdmin(t *testing.T) {
	h, adminID := mustLoginNewAdmin(t)
	w := testSrv.PATCH(userPath(adminID), map[string]any{"is_platform_admin": false}, h)
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestUser_Update_InvalidPayload(t *testing.T) {
	userID := mustCreateUser(t, testhelper.UniqueEmail("upd"))
	for name, body := range map[string]map[string]any{
		"email tidak valid": {"email": "bukan-email"},
		"nama kosong":       {"name": "  "},
		"password pendek":   {"password": "123"},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, 422, testSrv.PATCH(userPath(userID), body, adminHeader()).Code)
		})
	}
}

func TestUser_Update_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.PATCH(userPath(uuid.NewString()), map[string]any{"name": "X"}, adminHeader()).Code)
}

func TestUser_Update_DuplicateEmail(t *testing.T) {
	taken := testhelper.UniqueEmail("taken")
	mustCreateUser(t, taken)
	userID := mustCreateUser(t, testhelper.UniqueEmail("upd"))
	assert.Equal(t, 409, testSrv.PATCH(userPath(userID), map[string]any{"email": taken}, adminHeader()).Code)
}

func TestUser_Update_Success(t *testing.T) {
	userID := mustCreateUser(t, testhelper.UniqueEmail("upd"))
	newEmail := testhelper.UniqueEmail("renamed")

	w := testSrv.PATCH(userPath(userID), map[string]any{
		"name": "Nama Baru", "email": newEmail, "is_platform_admin": true, "password": "PasswordBaru123",
	}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got userData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "Nama Baru", got.Name)
	assert.Equal(t, newEmail, got.Email)
	assert.True(t, got.IsPlatformAdmin)
	assert.Equal(t, 1, countUserEvents(t, "user.updated", userID))

	// Password hasil reset admin bisa dipakai login.
	code := testhelper.UniqueCode("upd")
	testhelper.MustSeedSolution(t, testSrv.DB, code, "active")
	w = testSrv.POST("/api/v1/auth/login", map[string]any{"email": newEmail, "password": "PasswordBaru123", "solution": code})
	assert.Equal(t, 200, w.Code, w.Body.String())
}

// ── Status ───────────────────────────────────────────────────────────────────

func TestUser_ChangeStatus_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.PATCH(userPath(uuid.NewString())+"/status", map[string]any{"status": "active"}).Code)
}

func TestUser_ChangeStatus_Forbidden_NonAdmin(t *testing.T) {
	tok, userID, _ := mustLoginNewUser(t)
	w := testSrv.PATCH(userPath(userID)+"/status", map[string]any{"status": "suspended"}, testhelper.BearerHeader(tok.AccessToken))
	assert.Equal(t, 403, w.Code)
}

func TestUser_ChangeStatus_Forbidden_Self(t *testing.T) {
	h, adminID := mustLoginNewAdmin(t)
	w := testSrv.PATCH(userPath(adminID)+"/status", map[string]any{"status": "suspended"}, h)
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestUser_ChangeStatus_InvalidPayload(t *testing.T) {
	userID := mustCreateUser(t, testhelper.UniqueEmail("st"))
	assert.Equal(t, 422, testSrv.PATCH(userPath(userID)+"/status", map[string]any{"status": "deleted"}, adminHeader()).Code)
	assert.Equal(t, 422, testSrv.PATCH(userPath(userID)+"/status", map[string]any{}, adminHeader()).Code)
}

func TestUser_ChangeStatus_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.PATCH(userPath(uuid.NewString())+"/status", map[string]any{"status": "active"}, adminHeader()).Code)
}

func TestUser_ChangeStatus_Success_TokenRejected(t *testing.T) {
	tok, userID, _ := mustLoginNewUser(t)
	h := testhelper.BearerHeader(tok.AccessToken)
	require.Equal(t, 200, testSrv.GET("/api/v1/users/me", h).Code)

	w := testSrv.PATCH(userPath(userID)+"/status", map[string]any{"status": "suspended"}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got userData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "suspended", got.Status)
	assert.Equal(t, 1, countUserEvents(t, "user.status_changed", userID))

	// Token yang sudah terbit langsung ditolak.
	assert.Equal(t, 401, testSrv.GET("/api/v1/users/me", h).Code)

	// Status sama → tidak ada event baru.
	require.Equal(t, 200, testSrv.PATCH(userPath(userID)+"/status", map[string]any{"status": "suspended"}, adminHeader()).Code)
	assert.Equal(t, 1, countUserEvents(t, "user.status_changed", userID))
}

// ── Me ───────────────────────────────────────────────────────────────────────

func TestUser_Me_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET("/api/v1/users/me").Code)
	assert.Equal(t, 401, testSrv.PATCH("/api/v1/users/me", map[string]any{"name": "X"}).Code)
}

func TestUser_Me_Forbidden_ServiceToken(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("me")), "users:read")
	h := testhelper.BearerHeader(mustServiceToken(t, c, ""))
	assert.Equal(t, 403, testSrv.GET("/api/v1/users/me", h).Code)
}

func TestUser_Me_Success(t *testing.T) {
	tok, userID, _ := mustLoginNewUser(t)
	w := testSrv.GET("/api/v1/users/me", testhelper.BearerHeader(tok.AccessToken))
	require.Equal(t, 200, w.Code, w.Body.String())
	var got userData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, userID, got.ID)
}

func TestUser_UpdateMe_InvalidPayload(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)
	h := testhelper.BearerHeader(tok.AccessToken)
	assert.Equal(t, 422, testSrv.PATCH("/api/v1/users/me", map[string]any{"name": " "}, h).Code)
	assert.Equal(t, 422, testSrv.PATCH("/api/v1/users/me", map[string]any{"password": "PasswordBaru123"}, h).Code, "tanpa current_password")
	assert.Equal(t, 422, testSrv.PATCH("/api/v1/users/me", map[string]any{
		"password": "PasswordBaru123", "current_password": "salah-sekali",
	}, h).Code, "current_password salah")
}

func TestUser_UpdateMe_Success(t *testing.T) {
	email := testhelper.UniqueEmail("me")
	userID := mustCreateUser(t, email)
	code := testhelper.UniqueCode("me")
	testhelper.MustSeedSolution(t, testSrv.DB, code, "active")
	h := testhelper.BearerHeader(mustLogin(t, email, code).AccessToken)

	w := testSrv.PATCH("/api/v1/users/me", map[string]any{
		"name": "Nama Saya", "password": "PasswordBaru123", "current_password": testPassword,
	}, h)
	require.Equal(t, 200, w.Code, w.Body.String())
	var got userData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "Nama Saya", got.Name)
	assert.Equal(t, 1, countUserEvents(t, "user.updated", userID))

	// Password lama tidak berlaku, yang baru berlaku.
	w = testSrv.POST("/api/v1/auth/login", map[string]any{"email": email, "password": testPassword, "solution": code})
	assert.Equal(t, 401, w.Code)
	w = testSrv.POST("/api/v1/auth/login", map[string]any{"email": email, "password": "PasswordBaru123", "solution": code})
	assert.Equal(t, 200, w.Code, w.Body.String())
}
