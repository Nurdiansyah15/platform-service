package api_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"platform-service/internal/testhelper"
)

func adminHeader() map[string]string {
	return testhelper.BearerHeader(testAdminTok)
}

// userHeader — token user biasa dari FakeAuthenticator (tanpa user di DB).
// Pakai mustLogin bila skenario butuh user & sesi nyata.
func userHeader() map[string]string {
	return testhelper.BearerHeader(testSrv.Auth.UserToken())
}

// mustCreateSolution membuat solution via API (sebagai platform admin) dan mengembalikan ID-nya.
func mustCreateSolution(t *testing.T, code string) string {
	t.Helper()
	w := testSrv.POST("/api/v1/solutions", map[string]any{
		"code": code,
		"name": "Solution " + code,
	}, adminHeader())
	require.Equal(t, 201, w.Code, "mustCreateSolution body=%s", w.Body.String())
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.NotEmpty(t, resp.Data.ID)
	return resp.Data.ID
}

// mustCreateUser membuat user aktif ber-password via API dan mengembalikan ID-nya.
func mustCreateUser(t *testing.T, email string) string {
	t.Helper()
	w := testSrv.POST("/api/v1/users", map[string]any{
		"email":    email,
		"name":     "User " + email,
		"password": testPassword,
	}, adminHeader())
	require.Equal(t, 201, w.Code, "mustCreateUser body=%s", w.Body.String())
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	return resp.Data.ID
}

type tokenPair struct {
	AccessToken  string  `json:"access_token"`
	TokenType    string  `json:"token_type"`
	ExpiresIn    int64   `json:"expires_in"`
	RefreshToken string  `json:"refresh_token"`
	SessionID    string  `json:"session_id"`
	TenantID     *string `json:"tenant_id"`
}

func decodeTokens(body []byte) tokenPair {
	var resp struct {
		Data tokenPair `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		panic("decodeTokens: " + err.Error())
	}
	return resp.Data
}

// mustLogin login sebagai email/testPassword ke solution tertentu.
func mustLogin(t *testing.T, email, solutionCode string) tokenPair {
	t.Helper()
	w := testSrv.POST("/api/v1/auth/login", map[string]any{
		"email": email, "password": testPassword, "solution": solutionCode,
	})
	require.Equal(t, 200, w.Code, "mustLogin body=%s", w.Body.String())
	return decodeTokens(w.Body.Bytes())
}

// mustLoginNewUser membuat user baru + solution baru lalu login. Mengembalikan
// token, ID user, dan code solution.
func mustLoginNewUser(t *testing.T) (tokenPair, string, string) {
	t.Helper()
	email := testhelper.UniqueEmail("user")
	userID := mustCreateUser(t, email)
	code := testhelper.UniqueCode("app")
	testhelper.MustSeedSolution(t, testSrv.DB, code, "active")
	return mustLogin(t, email, code), userID, code
}

func upper(s string) string { return strings.ToUpper(s) }
func lower(s string) string { return strings.ToLower(s) }

type clientCredentials struct {
	ID           string   `json:"id"`
	SolutionID   string   `json:"solution_id"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	Scopes       []string `json:"scopes"`
	Status       string   `json:"status"`
}

// mustCreateClient membuat solution client via API (admin) dan mengembalikan kredensialnya.
func mustCreateClient(t *testing.T, solutionID string, scopes ...string) clientCredentials {
	t.Helper()
	w := testSrv.POST("/api/v1/solutions/"+solutionID+"/clients", map[string]any{
		"name": "Backend " + testhelper.UniqueCode("c"), "scopes": scopes,
	}, adminHeader())
	require.Equal(t, 201, w.Code, "mustCreateClient body=%s", w.Body.String())
	var got clientCredentials
	testhelper.DecodeBody(t, w, &got)
	return got
}

// mustServiceToken menukar kredensial client menjadi access token service (OAuth2).
func mustServiceToken(t *testing.T, c clientCredentials, scope string) string {
	t.Helper()
	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {c.ClientID}, "client_secret": {c.ClientSecret}}
	if scope != "" {
		form.Set("scope", scope)
	}
	w := testSrv.POSTForm("/api/v1/oauth/token", form)
	require.Equal(t, 200, w.Code, "mustServiceToken body=%s", w.Body.String())
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.AccessToken
}
