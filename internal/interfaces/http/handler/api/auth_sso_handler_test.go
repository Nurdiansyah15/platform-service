package api_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/testhelper"
)

type ssoSolution struct {
	ID          string
	Code        string
	LaunchURL   string
	RedirectURI string
}

// mustCreateSSOSolution membuat solution (admin, via API) lengkap dengan launch_url & redirect_uris.
func mustCreateSSOSolution(t *testing.T) ssoSolution {
	t.Helper()
	code := testhelper.UniqueCode("sso")
	s := ssoSolution{
		Code:        code,
		LaunchURL:   "https://" + code + ".example.test/",
		RedirectURI: "https://" + code + ".example.test/auth/callback",
	}
	w := testSrv.POST("/api/v1/solutions", map[string]any{
		"code": code, "name": "Solution " + code,
		"launch_url": s.LaunchURL, "redirect_uris": []string{s.RedirectURI},
	}, adminHeader())
	require.Equal(t, 201, w.Code, "mustCreateSSOSolution body=%s", w.Body.String())
	var got struct {
		ID string `json:"id"`
	}
	testhelper.DecodeBody(t, w, &got)
	s.ID = got.ID
	return s
}

// pkcePair membuat code_verifier acak dan code_challenge S256-nya.
func pkcePair(t *testing.T) (verifier, challenge string) {
	t.Helper()
	b := make([]byte, 32)
	_, err := rand.Read(b)
	require.NoError(t, err)
	verifier = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

// mustConsoleMember membuat user baru yang login ke console dengan company aktif
// (buat company → switch-company). Mengembalikan token console, ID user, dan ID company.
func mustConsoleMember(t *testing.T) (tokenPair, string, string) {
	t.Helper()
	email := testhelper.UniqueEmail("sso")
	userID := mustCreateUser(t, email)
	tok := mustLogin(t, email, testConsoleSolution)
	companyID := mustCreateCompany(t, testhelper.BearerHeader(tok.AccessToken))
	return mustSwitchCompany(t, tok.AccessToken, companyID), userID, companyID
}

func authorizeBody(s ssoSolution, challenge string) map[string]any {
	return map[string]any{
		"solution": s.Code, "redirect_uri": s.RedirectURI,
		"code_challenge": challenge, "code_challenge_method": "S256", "state": "xyz-state",
	}
}

type authorizeData struct {
	RedirectTo string `json:"redirect_to"`
	Code       string `json:"code"`
}

func mustAuthorize(t *testing.T, accessToken string, body map[string]any) authorizeData {
	t.Helper()
	w := testSrv.POST("/api/v1/auth/authorize", body, testhelper.BearerHeader(accessToken))
	require.Equal(t, 200, w.Code, "mustAuthorize body=%s", w.Body.String())
	var got authorizeData
	testhelper.DecodeBody(t, w, &got)
	return got
}

func exchangeBody(code, verifier, redirectURI string) map[string]any {
	return map[string]any{"code": code, "code_verifier": verifier, "redirect_uri": redirectURI}
}

// ── Authorize ────────────────────────────────────────────────────────────────

func TestAuth_Authorize_Unauthenticated(t *testing.T) {
	w := testSrv.POST("/api/v1/auth/authorize", map[string]any{})
	assert.Equal(t, 401, w.Code)
}

func TestAuth_Authorize_ServiceToken_Forbidden(t *testing.T) {
	w := testSrv.POST("/api/v1/auth/authorize", map[string]any{},
		testhelper.BearerHeader(testSrv.Auth.ServiceToken(uuid.NewString())))
	assert.Equal(t, 403, w.Code)
}

func TestAuth_Authorize_InvalidPayload(t *testing.T) {
	tok, _, _ := mustConsoleMember(t)
	w := testSrv.POST("/api/v1/auth/authorize", map[string]any{"solution": "x"}, testhelper.BearerHeader(tok.AccessToken))
	assert.Equal(t, 422, w.Code, w.Body.String())
}

func TestAuth_Authorize_NonConsoleSession_Forbidden(t *testing.T) {
	s := mustCreateSSOSolution(t)
	tok, _, _ := mustLoginNewUser(t) // sesi solution lain, bukan console
	_, challenge := pkcePair(t)

	w := testSrv.POST("/api/v1/auth/authorize", authorizeBody(s, challenge), testhelper.BearerHeader(tok.AccessToken))
	require.Equal(t, 403, w.Code, w.Body.String())
	assert.Equal(t, "Only a platform console session can sign in to other solutions.", errorMessage(t, testhelper.DecodeError(t, w)))
}

func TestAuth_Authorize_UnregisteredRedirectURI(t *testing.T) {
	s := mustCreateSSOSolution(t)
	tok, _, _ := mustConsoleMember(t)
	_, challenge := pkcePair(t)
	body := authorizeBody(s, challenge)
	body["redirect_uri"] = "https://evil.example.test/auth/callback"

	w := testSrv.POST("/api/v1/auth/authorize", body, testhelper.BearerHeader(tok.AccessToken))
	require.Equal(t, 422, w.Code, w.Body.String())
	assert.Equal(t, "redirect_uri is not registered for this solution.", errorMessage(t, testhelper.DecodeError(t, w)))
}

func TestAuth_Authorize_UnknownSolution(t *testing.T) {
	tok, _, _ := mustConsoleMember(t)
	_, challenge := pkcePair(t)
	body := authorizeBody(ssoSolution{Code: "no-such-solution", RedirectURI: "https://x.example.test/cb"}, challenge)

	w := testSrv.POST("/api/v1/auth/authorize", body, testhelper.BearerHeader(tok.AccessToken))
	assert.Equal(t, 422, w.Code, w.Body.String())
}

func TestAuth_Authorize_CompanyNotMember_Forbidden(t *testing.T) {
	s := mustCreateSSOSolution(t)
	tok, _, _ := mustConsoleMember(t)
	_, _, otherCompany := mustConsoleMember(t)
	_, challenge := pkcePair(t)
	body := authorizeBody(s, challenge)
	body["company_id"] = otherCompany

	w := testSrv.POST("/api/v1/auth/authorize", body, testhelper.BearerHeader(tok.AccessToken))
	require.Equal(t, 403, w.Code, w.Body.String())
	assert.Equal(t, "This company cannot be selected: you must be an active member, the company must be active, and it must subscribe to this solution.", errorMessage(t, testhelper.DecodeError(t, w)))
}

func TestAuth_Authorize_Success(t *testing.T) {
	s := mustCreateSSOSolution(t)
	tok, userID, companyID := mustConsoleMember(t)
	_, challenge := pkcePair(t)

	got := mustAuthorize(t, tok.AccessToken, authorizeBody(s, challenge))
	require.NotEmpty(t, got.Code)

	u, err := url.Parse(got.RedirectTo)
	require.NoError(t, err)
	assert.Equal(t, s.RedirectURI, u.Scheme+"://"+u.Host+u.Path)
	assert.Equal(t, got.Code, u.Query().Get("code"))
	assert.Equal(t, "xyz-state", u.Query().Get("state"))

	// Hanya hash kode yang disimpan; company aktif console terbawa.
	var storedUser, storedCompany, parent string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT user_id, company_id, parent_session_id FROM auth_authorization_codes WHERE solution_id = $1`, s.ID,
	).Scan(&storedUser, &storedCompany, &parent))
	assert.Equal(t, userID, storedUser)
	assert.Equal(t, companyID, storedCompany)
	assert.Equal(t, tok.SessionID, parent)

	var raw int
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM auth_authorization_codes WHERE code_hash = $1`, got.Code).Scan(&raw))
	assert.Zero(t, raw)
}

// ── Token exchange ───────────────────────────────────────────────────────────

func TestAuth_TokenExchange_InvalidPayload(t *testing.T) {
	w := testSrv.POST("/api/v1/auth/token-exchange", map[string]any{"code": "abc"})
	assert.Equal(t, 422, w.Code, w.Body.String())
}

func TestAuth_TokenExchange_UnknownCode(t *testing.T) {
	verifier, _ := pkcePair(t)
	w := testSrv.POST("/api/v1/auth/token-exchange", exchangeBody("no-such-code", verifier, "https://x.example.test/cb"))
	require.Equal(t, 400, w.Code, w.Body.String())
	assert.Equal(t, "The authorization code is invalid, expired, or already used. Please sign in again from the console.", errorMessage(t, testhelper.DecodeError(t, w)))
}

func TestAuth_TokenExchange_Success(t *testing.T) {
	s := mustCreateSSOSolution(t)
	consoleTok, userID, companyID := mustConsoleMember(t)
	verifier, challenge := pkcePair(t)
	code := mustAuthorize(t, consoleTok.AccessToken, authorizeBody(s, challenge)).Code

	w := testSrv.POST("/api/v1/auth/token-exchange", exchangeBody(code, verifier, s.RedirectURI))
	require.Equal(t, 200, w.Code, w.Body.String())
	tok := decodeTokens(w.Body.Bytes())
	require.NotEmpty(t, tok.AccessToken)
	require.NotEmpty(t, tok.RefreshToken)
	require.NotNil(t, tok.CompanyID)
	assert.Equal(t, companyID, *tok.CompanyID)
	assert.NotEqual(t, consoleTok.SessionID, tok.SessionID)

	// Token baru ber-aud solution tujuan dan langsung bisa dipakai.
	w = testSrv.GET("/api/v1/auth/me", testhelper.BearerHeader(tok.AccessToken))
	require.Equal(t, 200, w.Code, w.Body.String())
	var me struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
		Session struct {
			Solution struct {
				Code string `json:"code"`
			} `json:"solution"`
			CompanyID *string `json:"company_id"`
		} `json:"session"`
	}
	testhelper.DecodeBody(t, w, &me)
	assert.Equal(t, userID, me.User.ID)
	assert.Equal(t, s.Code, me.Session.Solution.Code)

	var parent string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT parent_session_id FROM auth_sessions WHERE id = $1`, tok.SessionID).Scan(&parent))
	assert.Equal(t, consoleTok.SessionID, parent)

	// Refresh berjalan seperti sesi login biasa.
	w = testSrv.POST("/api/v1/auth/refresh", map[string]any{"refresh_token": tok.RefreshToken})
	assert.Equal(t, 200, w.Code, w.Body.String())
}

func TestAuth_TokenExchange_CodeReused(t *testing.T) {
	s := mustCreateSSOSolution(t)
	consoleTok, _, _ := mustConsoleMember(t)
	verifier, challenge := pkcePair(t)
	code := mustAuthorize(t, consoleTok.AccessToken, authorizeBody(s, challenge)).Code

	w := testSrv.POST("/api/v1/auth/token-exchange", exchangeBody(code, verifier, s.RedirectURI))
	require.Equal(t, 200, w.Code, w.Body.String())

	w = testSrv.POST("/api/v1/auth/token-exchange", exchangeBody(code, verifier, s.RedirectURI))
	require.Equal(t, 400, w.Code, w.Body.String())
	assert.Equal(t, "The authorization code is invalid, expired, or already used. Please sign in again from the console.", errorMessage(t, testhelper.DecodeError(t, w)))
}

func TestAuth_TokenExchange_WrongVerifier(t *testing.T) {
	s := mustCreateSSOSolution(t)
	consoleTok, _, _ := mustConsoleMember(t)
	_, challenge := pkcePair(t)
	otherVerifier, _ := pkcePair(t)
	code := mustAuthorize(t, consoleTok.AccessToken, authorizeBody(s, challenge)).Code

	w := testSrv.POST("/api/v1/auth/token-exchange", exchangeBody(code, otherVerifier, s.RedirectURI))
	assert.Equal(t, 400, w.Code, w.Body.String())
}

func TestAuth_TokenExchange_RedirectURIMismatch(t *testing.T) {
	s := mustCreateSSOSolution(t)
	consoleTok, _, _ := mustConsoleMember(t)
	verifier, challenge := pkcePair(t)
	code := mustAuthorize(t, consoleTok.AccessToken, authorizeBody(s, challenge)).Code

	w := testSrv.POST("/api/v1/auth/token-exchange", exchangeBody(code, verifier, s.RedirectURI+"?x=1"))
	assert.Equal(t, 400, w.Code, w.Body.String())
}

func TestAuth_TokenExchange_Expired(t *testing.T) {
	s := mustCreateSSOSolution(t)
	consoleTok, _, _ := mustConsoleMember(t)
	verifier, challenge := pkcePair(t)
	code := mustAuthorize(t, consoleTok.AccessToken, authorizeBody(s, challenge)).Code
	_, err := testSrv.DB.ExecContext(context.Background(),
		`UPDATE auth_authorization_codes SET expires_at = NOW() - INTERVAL '1 second' WHERE solution_id = $1`, s.ID)
	require.NoError(t, err)

	w := testSrv.POST("/api/v1/auth/token-exchange", exchangeBody(code, verifier, s.RedirectURI))
	assert.Equal(t, 400, w.Code, w.Body.String())
}

func TestAuth_TokenExchange_UserSuspendedAfterAuthorize(t *testing.T) {
	s := mustCreateSSOSolution(t)
	consoleTok, userID, _ := mustConsoleMember(t)
	verifier, challenge := pkcePair(t)
	code := mustAuthorize(t, consoleTok.AccessToken, authorizeBody(s, challenge)).Code
	testhelper.SetUserStatus(t, testSrv.DB, userID, "suspended")

	w := testSrv.POST("/api/v1/auth/token-exchange", exchangeBody(code, verifier, s.RedirectURI))
	assert.Equal(t, 400, w.Code, w.Body.String())
}

func TestAuth_TokenExchange_ConsoleLoggedOutAfterAuthorize(t *testing.T) {
	s := mustCreateSSOSolution(t)
	consoleTok, _, _ := mustConsoleMember(t)
	verifier, challenge := pkcePair(t)
	code := mustAuthorize(t, consoleTok.AccessToken, authorizeBody(s, challenge)).Code
	require.Equal(t, 200, testSrv.POST("/api/v1/auth/logout", nil, testhelper.BearerHeader(consoleTok.AccessToken)).Code)

	w := testSrv.POST("/api/v1/auth/token-exchange", exchangeBody(code, verifier, s.RedirectURI))
	assert.Equal(t, 400, w.Code, w.Body.String())
}

func TestAuth_Logout_ConsoleRevokesSSOSessions(t *testing.T) {
	s := mustCreateSSOSolution(t)
	consoleTok, _, _ := mustConsoleMember(t)
	verifier, challenge := pkcePair(t)
	code := mustAuthorize(t, consoleTok.AccessToken, authorizeBody(s, challenge)).Code
	w := testSrv.POST("/api/v1/auth/token-exchange", exchangeBody(code, verifier, s.RedirectURI))
	require.Equal(t, 200, w.Code, w.Body.String())
	child := decodeTokens(w.Body.Bytes())

	require.Equal(t, 200, testSrv.POST("/api/v1/auth/logout", nil, testhelper.BearerHeader(consoleTok.AccessToken)).Code)

	w = testSrv.GET("/api/v1/auth/me", testhelper.BearerHeader(child.AccessToken))
	assert.Equal(t, 401, w.Code, w.Body.String())
}

func TestAuth_Logout_SSOSessionKeepsConsole(t *testing.T) {
	s := mustCreateSSOSolution(t)
	consoleTok, _, _ := mustConsoleMember(t)
	verifier, challenge := pkcePair(t)
	code := mustAuthorize(t, consoleTok.AccessToken, authorizeBody(s, challenge)).Code
	w := testSrv.POST("/api/v1/auth/token-exchange", exchangeBody(code, verifier, s.RedirectURI))
	require.Equal(t, 200, w.Code, w.Body.String())
	child := decodeTokens(w.Body.Bytes())

	require.Equal(t, 200, testSrv.POST("/api/v1/auth/logout", nil, testhelper.BearerHeader(child.AccessToken)).Code)

	w = testSrv.GET("/api/v1/auth/me", testhelper.BearerHeader(consoleTok.AccessToken))
	assert.Equal(t, 200, w.Code, w.Body.String())
}

// ── Apps ─────────────────────────────────────────────────────────────────────

type appItem struct {
	Code      string `json:"code"`
	LaunchURL string `json:"launch_url"`
	IsCurrent bool   `json:"is_current"`
}

func findApp(items []appItem, code string) *appItem {
	for i := range items {
		if items[i].Code == code {
			return &items[i]
		}
	}
	return nil
}

func TestAuth_Apps_Unauthenticated(t *testing.T) {
	w := testSrv.GET("/api/v1/auth/apps")
	assert.Equal(t, 401, w.Code)
}

func TestAuth_Apps_ServiceToken_Forbidden(t *testing.T) {
	w := testSrv.GET("/api/v1/auth/apps", testhelper.BearerHeader(testSrv.Auth.ServiceToken(uuid.NewString())))
	assert.Equal(t, 403, w.Code)
}

func TestAuth_Apps_WithoutCompany_Empty(t *testing.T) {
	mustCreateSSOSolution(t)
	tok, _, _ := mustLoginNewUser(t)

	w := testSrv.GET("/api/v1/auth/apps", testhelper.BearerHeader(tok.AccessToken))
	require.Equal(t, 200, w.Code, w.Body.String())
	var items []appItem
	testhelper.DecodeBody(t, w, &items)
	assert.Empty(t, items)
}

func TestAuth_Apps_Success(t *testing.T) {
	s := mustCreateSSOSolution(t)
	noLaunch := testhelper.UniqueCode("nolaunch")
	testhelper.MustSeedSolution(t, testSrv.DB, noLaunch, "active")
	consoleTok, _, _ := mustConsoleMember(t)

	w := testSrv.GET("/api/v1/auth/apps", testhelper.BearerHeader(consoleTok.AccessToken))
	require.Equal(t, 200, w.Code, w.Body.String())
	var items []appItem
	testhelper.DecodeBody(t, w, &items)
	app := findApp(items, s.Code)
	require.NotNil(t, app)
	assert.Equal(t, s.LaunchURL, app.LaunchURL)
	assert.False(t, app.IsCurrent)
	assert.Nil(t, findApp(items, noLaunch), "solution tanpa launch_url disembunyikan")

	// Dari dalam solution (token hasil SSO), solution itu ditandai is_current.
	verifier, challenge := pkcePair(t)
	code := mustAuthorize(t, consoleTok.AccessToken, authorizeBody(s, challenge)).Code
	w = testSrv.POST("/api/v1/auth/token-exchange", exchangeBody(code, verifier, s.RedirectURI))
	require.Equal(t, 200, w.Code, w.Body.String())
	child := decodeTokens(w.Body.Bytes())

	w = testSrv.GET("/api/v1/auth/apps", testhelper.BearerHeader(child.AccessToken))
	require.Equal(t, 200, w.Code, w.Body.String())
	items = nil
	testhelper.DecodeBody(t, w, &items)
	app = findApp(items, s.Code)
	require.NotNil(t, app)
	assert.True(t, app.IsCurrent)
}

func TestAuth_Apps_PlatformAdminWithoutCompany(t *testing.T) {
	s := mustCreateSSOSolution(t)
	w := testSrv.GET("/api/v1/auth/apps", adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var items []appItem
	testhelper.DecodeBody(t, w, &items)
	assert.NotNil(t, findApp(items, s.Code))
}
