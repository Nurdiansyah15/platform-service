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

func errorMessage(t *testing.T, body testhelper.ErrorBody) string {
	t.Helper()
	var msg string
	require.NoError(t, json.Unmarshal(body.Errors, &msg))
	return msg
}

func loginBody(email, password, solution string) map[string]any {
	return map[string]any{"email": email, "password": password, "solution": solution}
}

// ── Login ────────────────────────────────────────────────────────────────────

func TestAuth_Login_InvalidPayload(t *testing.T) {
	w := testSrv.POST("/api/v1/auth/login", map[string]any{"email": "a@b.co"})
	require.Equal(t, 422, w.Code, w.Body.String())

	var fields map[string]string
	require.NoError(t, json.Unmarshal(testhelper.DecodeError(t, w).Errors, &fields))
	assert.Contains(t, fields, "password")
	assert.Contains(t, fields, "solution")
}

func TestAuth_Login_WrongPassword(t *testing.T) {
	email := testhelper.UniqueEmail("wrongpw")
	mustCreateUser(t, email)

	w := testSrv.POST("/api/v1/auth/login", loginBody(email, "salah-password", testConsoleSolution))
	require.Equal(t, 401, w.Code, w.Body.String())
	assert.Equal(t, "Invalid email or password.", errorMessage(t, testhelper.DecodeError(t, w)))
}

func TestAuth_Login_UnknownEmail_SameResponseAsWrongPassword(t *testing.T) {
	w := testSrv.POST("/api/v1/auth/login", loginBody(testhelper.UniqueEmail("ghost"), testPassword, testConsoleSolution))
	require.Equal(t, 401, w.Code, w.Body.String())
	assert.Equal(t, "Invalid email or password.", errorMessage(t, testhelper.DecodeError(t, w)))
}

func TestAuth_Login_UserWithoutPassword(t *testing.T) {
	email := testhelper.UniqueEmail("nopw")
	w := testSrv.POST("/api/v1/users", map[string]any{"email": email, "name": "Tanpa Password"}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())

	w = testSrv.POST("/api/v1/auth/login", loginBody(email, testPassword, testConsoleSolution))
	assert.Equal(t, 401, w.Code, w.Body.String())
}

func TestAuth_Login_SuspendedUser_Forbidden(t *testing.T) {
	email := testhelper.UniqueEmail("suspended")
	userID := mustCreateUser(t, email)
	testhelper.SetUserStatus(t, testSrv.DB, userID, "suspended")

	w := testSrv.POST("/api/v1/auth/login", loginBody(email, testPassword, testConsoleSolution))
	require.Equal(t, 403, w.Code, w.Body.String())
	assert.Equal(t, "Your account is not active.", errorMessage(t, testhelper.DecodeError(t, w)))
}

func TestAuth_Login_SuspendedUserWrongPassword_StillUnauthorized(t *testing.T) {
	email := testhelper.UniqueEmail("suspended-wrongpw")
	userID := mustCreateUser(t, email)
	testhelper.SetUserStatus(t, testSrv.DB, userID, "suspended")

	w := testSrv.POST("/api/v1/auth/login", loginBody(email, "salah-password", testConsoleSolution))
	assert.Equal(t, 401, w.Code, "status akun tidak boleh bocor tanpa password benar")
}

func TestAuth_Login_UnknownSolution(t *testing.T) {
	email := testhelper.UniqueEmail("nosol")
	mustCreateUser(t, email)

	w := testSrv.POST("/api/v1/auth/login", loginBody(email, testPassword, "solution-tidak-ada"))
	assert.Equal(t, 422, w.Code, w.Body.String())
}

func TestAuth_Login_InactiveSolution(t *testing.T) {
	email := testhelper.UniqueEmail("inactivesol")
	mustCreateUser(t, email)
	code := testhelper.UniqueCode("off")
	testhelper.MustSeedSolution(t, testSrv.DB, code, "inactive")

	w := testSrv.POST("/api/v1/auth/login", loginBody(email, testPassword, code))
	assert.Equal(t, 422, w.Code, w.Body.String())
}

func TestAuth_Login_Success(t *testing.T) {
	email := testhelper.UniqueEmail("ok")
	userID := mustCreateUser(t, email)

	// Email case-insensitive.
	w := testSrv.POST("/api/v1/auth/login", loginBody("  "+upper(email)+" ", testPassword, testConsoleSolution),
		map[string]string{"User-Agent": "platform-test/1.0"})
	require.Equal(t, 200, w.Code, w.Body.String())

	tok := decodeTokens(w.Body.Bytes())
	assert.NotEmpty(t, tok.AccessToken)
	assert.Equal(t, "Bearer", tok.TokenType)
	assert.InDelta(t, 900, tok.ExpiresIn, 2)
	assert.NotEmpty(t, tok.RefreshToken)
	assert.Nil(t, tok.TenantID)

	var (
		dbUserID, dbHash string
		dbUserAgent      *string
	)
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT user_id, refresh_token_hash, user_agent FROM auth_sessions WHERE id = $1`, tok.SessionID,
	).Scan(&dbUserID, &dbHash, &dbUserAgent))
	assert.Equal(t, userID, dbUserID)
	assert.NotEqual(t, tok.RefreshToken, dbHash, "refresh token mentah tidak boleh disimpan")
	assert.Len(t, dbHash, 64)
	require.NotNil(t, dbUserAgent)
	assert.Equal(t, "platform-test/1.0", *dbUserAgent)
}

// ── Me ───────────────────────────────────────────────────────────────────────

func TestAuth_Me_Unauthenticated(t *testing.T) {
	w := testSrv.GET("/api/v1/auth/me")
	assert.Equal(t, 401, w.Code)
}

func TestAuth_Me_ServiceToken_Forbidden(t *testing.T) {
	w := testSrv.GET("/api/v1/auth/me", testhelper.BearerHeader(testSrv.Auth.ServiceToken(uuid.NewString(), "users:read")))
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestAuth_Me_Success(t *testing.T) {
	tok, userID, code := mustLoginNewUser(t)

	w := testSrv.GET("/api/v1/auth/me", testhelper.BearerHeader(tok.AccessToken))
	require.Equal(t, 200, w.Code, w.Body.String())

	var got struct {
		User struct {
			ID              string `json:"id"`
			Email           string `json:"email"`
			Status          string `json:"status"`
			IsPlatformAdmin bool   `json:"is_platform_admin"`
			HasPassword     bool   `json:"has_password"`
		} `json:"user"`
		Session struct {
			ID       string `json:"id"`
			Solution struct {
				Code string `json:"code"`
			} `json:"solution"`
			TenantID *string `json:"tenant_id"`
		} `json:"session"`
	}
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, userID, got.User.ID)
	assert.Equal(t, "active", got.User.Status)
	assert.False(t, got.User.IsPlatformAdmin)
	assert.True(t, got.User.HasPassword)
	assert.Equal(t, tok.SessionID, got.Session.ID)
	assert.Equal(t, code, got.Session.Solution.Code)
	assert.Nil(t, got.Session.TenantID)
	assert.NotContains(t, w.Body.String(), "password_hash")
}

func TestAuth_Me_PlatformAdminFlagFromDB(t *testing.T) {
	w := testSrv.GET("/api/v1/auth/me", adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got struct {
		User struct {
			IsPlatformAdmin bool `json:"is_platform_admin"`
		} `json:"user"`
	}
	testhelper.DecodeBody(t, w, &got)
	assert.True(t, got.User.IsPlatformAdmin)
}

func TestAuth_AccessToken_RejectedAfterUserSuspended(t *testing.T) {
	tok, userID, _ := mustLoginNewUser(t)
	testhelper.SetUserStatus(t, testSrv.DB, userID, "suspended")

	w := testSrv.GET("/api/v1/auth/me", testhelper.BearerHeader(tok.AccessToken))
	assert.Equal(t, 401, w.Code, w.Body.String())
}

func TestAuth_AccessToken_RejectedAfterSolutionDeactivated(t *testing.T) {
	tok, _, code := mustLoginNewUser(t)
	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE solutions SET status = 'inactive' WHERE code = $1`, code)
	require.NoError(t, err)

	w := testSrv.GET("/api/v1/auth/me", testhelper.BearerHeader(tok.AccessToken))
	assert.Equal(t, 401, w.Code, w.Body.String())
}

// ── Refresh ──────────────────────────────────────────────────────────────────

func TestAuth_Refresh_InvalidPayload(t *testing.T) {
	w := testSrv.POST("/api/v1/auth/refresh", map[string]any{})
	assert.Equal(t, 422, w.Code, w.Body.String())
}

func TestAuth_Refresh_UnknownToken(t *testing.T) {
	w := testSrv.POST("/api/v1/auth/refresh", map[string]any{"refresh_token": "tidak-dikenal"})
	require.Equal(t, 401, w.Code, w.Body.String())
	assert.Equal(t, "Refresh token is invalid or expired.", errorMessage(t, testhelper.DecodeError(t, w)))
}

func TestAuth_Refresh_RotatesToken(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)

	w := testSrv.POST("/api/v1/auth/refresh", map[string]any{"refresh_token": tok.RefreshToken})
	require.Equal(t, 200, w.Code, w.Body.String())
	next := decodeTokens(w.Body.Bytes())
	assert.NotEqual(t, tok.RefreshToken, next.RefreshToken)
	assert.Equal(t, tok.SessionID, next.SessionID, "refresh tetap di sesi yang sama")

	// Access token baru berlaku.
	w = testSrv.GET("/api/v1/auth/me", testhelper.BearerHeader(next.AccessToken))
	assert.Equal(t, 200, w.Code, w.Body.String())

	// Refresh token lama tidak bisa dipakai lagi.
	w = testSrv.POST("/api/v1/auth/refresh", map[string]any{"refresh_token": tok.RefreshToken})
	assert.Equal(t, 401, w.Code, w.Body.String())

	// Refresh token baru masih bisa.
	w = testSrv.POST("/api/v1/auth/refresh", map[string]any{"refresh_token": next.RefreshToken})
	assert.Equal(t, 200, w.Code, w.Body.String())
}

func TestAuth_Refresh_ExpiredSession(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)
	_, err := testSrv.DB.ExecContext(context.Background(),
		`UPDATE auth_sessions SET expires_at = NOW() - INTERVAL '1 minute' WHERE id = $1`, tok.SessionID)
	require.NoError(t, err)

	w := testSrv.POST("/api/v1/auth/refresh", map[string]any{"refresh_token": tok.RefreshToken})
	assert.Equal(t, 401, w.Code, w.Body.String())
}

func TestAuth_Refresh_SuspendedUser(t *testing.T) {
	tok, userID, _ := mustLoginNewUser(t)
	testhelper.SetUserStatus(t, testSrv.DB, userID, "suspended")

	w := testSrv.POST("/api/v1/auth/refresh", map[string]any{"refresh_token": tok.RefreshToken})
	assert.Equal(t, 401, w.Code, w.Body.String())
}

// ── Logout ───────────────────────────────────────────────────────────────────

func TestAuth_Logout_Unauthenticated(t *testing.T) {
	w := testSrv.POST("/api/v1/auth/logout", nil)
	assert.Equal(t, 401, w.Code)
}

func TestAuth_Logout_ServiceToken_Forbidden(t *testing.T) {
	w := testSrv.POST("/api/v1/auth/logout", nil, testhelper.BearerHeader(testSrv.Auth.ServiceToken(uuid.NewString())))
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestAuth_Logout_RevokesSession(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)

	w := testSrv.POST("/api/v1/auth/logout", nil, testhelper.BearerHeader(tok.AccessToken))
	require.Equal(t, 200, w.Code, w.Body.String())

	var revoked bool
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT revoked_at IS NOT NULL FROM auth_sessions WHERE id = $1`, tok.SessionID).Scan(&revoked))
	assert.True(t, revoked)

	// Platform menolak access token dari sesi yang sudah logout.
	w = testSrv.GET("/api/v1/auth/me", testhelper.BearerHeader(tok.AccessToken))
	assert.Equal(t, 401, w.Code, w.Body.String())

	// Refresh token sesi tersebut juga mati.
	w = testSrv.POST("/api/v1/auth/refresh", map[string]any{"refresh_token": tok.RefreshToken})
	assert.Equal(t, 401, w.Code, w.Body.String())
}

func TestAuth_Logout_OtherSessionsUnaffected(t *testing.T) {
	email := testhelper.UniqueEmail("multi")
	mustCreateUser(t, email)
	first := mustLogin(t, email, testConsoleSolution)
	second := mustLogin(t, email, testConsoleSolution)

	w := testSrv.POST("/api/v1/auth/logout", nil, testhelper.BearerHeader(first.AccessToken))
	require.Equal(t, 200, w.Code, w.Body.String())

	w = testSrv.GET("/api/v1/auth/me", testhelper.BearerHeader(second.AccessToken))
	assert.Equal(t, 200, w.Code, w.Body.String())
}
