package api_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/testhelper"
)

const oauthTokenPath = "/api/v1/oauth/token"

func oauthForm(clientID, secret string) url.Values {
	return url.Values{"grant_type": {"client_credentials"}, "client_id": {clientID}, "client_secret": {secret}}
}

type oauthErr struct {
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

func decodeOAuthError(t *testing.T, body []byte) oauthErr {
	t.Helper()
	var e oauthErr
	require.NoError(t, json.Unmarshal(body, &e), string(body))
	return e
}

func basicAuth(id, secret string) map[string]string {
	return map[string]string{"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(id+":"+secret))}
}

// jwtClaims men-decode payload JWT tanpa verifikasi (verifikasi diuji di jwttoken).
func jwtClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]any
	require.NoError(t, json.Unmarshal(raw, &claims))
	return claims
}

// ── Grant & kredensial ───────────────────────────────────────────────────────

func TestOAuth_Token_MissingGrantType(t *testing.T) {
	w := testSrv.POSTForm(oauthTokenPath, url.Values{})
	require.Equal(t, 400, w.Code, w.Body.String())
	assert.Equal(t, "invalid_request", decodeOAuthError(t, w.Body.Bytes()).Error)
}

func TestOAuth_Token_UnsupportedGrantType(t *testing.T) {
	w := testSrv.POSTForm(oauthTokenPath, url.Values{"grant_type": {"password"}})
	require.Equal(t, 400, w.Code, w.Body.String())
	assert.Equal(t, "unsupported_grant_type", decodeOAuthError(t, w.Body.Bytes()).Error)
}

func TestOAuth_Token_MissingCredentials(t *testing.T) {
	w := testSrv.POSTForm(oauthTokenPath, url.Values{"grant_type": {"client_credentials"}})
	require.Equal(t, 401, w.Code, w.Body.String())
	assert.Equal(t, "invalid_client", decodeOAuthError(t, w.Body.Bytes()).Error)
}

func TestOAuth_Token_UnknownClient(t *testing.T) {
	w := testSrv.POSTForm(oauthTokenPath, oauthForm("cli_000000000000000000000000", "x"))
	require.Equal(t, 401, w.Code, w.Body.String())
	assert.Equal(t, "invalid_client", decodeOAuthError(t, w.Body.Bytes()).Error)
}

func TestOAuth_Token_WrongSecret(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("oa")))
	w := testSrv.POSTForm(oauthTokenPath, oauthForm(c.ClientID, "salah"))
	require.Equal(t, 401, w.Code, w.Body.String())
	assert.Equal(t, "invalid_client", decodeOAuthError(t, w.Body.Bytes()).Error)
}

func TestOAuth_Token_WrongSecretViaBasic_SetsWWWAuthenticate(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("oa")))
	w := testSrv.POSTForm(oauthTokenPath, url.Values{"grant_type": {"client_credentials"}}, basicAuth(c.ClientID, "salah"))
	require.Equal(t, 401, w.Code, w.Body.String())
	assert.Equal(t, `Basic realm="platform"`, w.Header().Get("WWW-Authenticate"))
}

func TestOAuth_Token_BothBasicAndBody_Rejected(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("oa")))
	w := testSrv.POSTForm(oauthTokenPath, oauthForm(c.ClientID, c.ClientSecret), basicAuth(c.ClientID, c.ClientSecret))
	require.Equal(t, 400, w.Code, w.Body.String())
	assert.Equal(t, "invalid_request", decodeOAuthError(t, w.Body.Bytes()).Error)
}

func TestOAuth_Token_RevokedClient(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("oa")))
	_, err := testSrv.DB.ExecContext(context.Background(),
		`UPDATE solution_clients SET status = 'revoked', revoked_at = NOW() WHERE id = $1`, c.ID)
	require.NoError(t, err)

	w := testSrv.POSTForm(oauthTokenPath, oauthForm(c.ClientID, c.ClientSecret))
	assert.Equal(t, 401, w.Code, w.Body.String())
}

func TestOAuth_Token_InactiveSolution(t *testing.T) {
	code := testhelper.UniqueCode("oa")
	c := mustCreateClient(t, mustCreateSolution(t, code))
	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE solutions SET status = 'inactive' WHERE code = $1`, code)
	require.NoError(t, err)

	w := testSrv.POSTForm(oauthTokenPath, oauthForm(c.ClientID, c.ClientSecret))
	assert.Equal(t, 401, w.Code, w.Body.String())
}

// ── Scope ────────────────────────────────────────────────────────────────────

func TestOAuth_Token_ScopeExceedsClient(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("oa")), "usage:write")
	form := oauthForm(c.ClientID, c.ClientSecret)
	form.Set("scope", "usage:write catalog:sync")

	w := testSrv.POSTForm(oauthTokenPath, form)
	require.Equal(t, 400, w.Code, w.Body.String())
	assert.Equal(t, "invalid_scope", decodeOAuthError(t, w.Body.Bytes()).Error)
}

func TestOAuth_Token_ScopeSubset(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("oa")), "usage:write", "authz:check")
	form := oauthForm(c.ClientID, c.ClientSecret)
	form.Set("scope", "authz:check")

	w := testSrv.POSTForm(oauthTokenPath, form)
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp struct {
		Scope string `json:"scope"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "authz:check", resp.Scope)
}

// ── Sukses ───────────────────────────────────────────────────────────────────

func TestOAuth_Token_Success_BodyCredentials(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("oa"))
	c := mustCreateClient(t, solID, "usage:write", "authz:check")

	w := testSrv.POSTForm(oauthTokenPath, oauthForm(c.ClientID, c.ClientSecret))
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotContains(t, resp, "status", "format RFC 6749, tanpa envelope")
	assert.Equal(t, "Bearer", resp["token_type"])
	assert.InDelta(t, 900, resp["expires_in"], 2)
	assert.Equal(t, "authz:check usage:write", resp["scope"])

	// Claim token sesuai spec §2.8.
	token := resp["access_token"].(string)
	claims := jwtClaims(t, token)
	assert.Equal(t, "service", claims["typ"])
	assert.Equal(t, "platform", claims["aud"])
	assert.Equal(t, c.ClientID, claims["sub"])
	assert.Equal(t, solID, claims["sol"])

	var lastUsed bool
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT last_used_at IS NOT NULL FROM solution_clients WHERE id = $1`, c.ID).Scan(&lastUsed))
	assert.True(t, lastUsed)
}

func TestOAuth_Token_Success_BasicAuth(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("oa")), "events:read")
	w := testSrv.POSTForm(oauthTokenPath, url.Values{"grant_type": {"client_credentials"}}, basicAuth(c.ClientID, c.ClientSecret))
	assert.Equal(t, 200, w.Code, w.Body.String())
}

// ── Token service di endpoint terproteksi ────────────────────────────────────

func TestOAuth_ServiceToken_AcceptedOnProtectedRoute(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("oa")), "catalog:sync")
	w := testSrv.GET("/api/v1/solutions", testhelper.BearerHeader(mustServiceToken(t, c, "")))
	assert.Equal(t, 200, w.Code, w.Body.String())
}

func TestOAuth_ServiceToken_RejectedAfterRevoke(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("oa")))
	token := mustServiceToken(t, c, "")
	_, err := testSrv.DB.ExecContext(context.Background(),
		`UPDATE solution_clients SET status = 'revoked', revoked_at = NOW() WHERE id = $1`, c.ID)
	require.NoError(t, err)

	w := testSrv.GET("/api/v1/solutions", testhelper.BearerHeader(token))
	assert.Equal(t, 401, w.Code, w.Body.String())
}
