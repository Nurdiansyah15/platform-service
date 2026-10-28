package api_test

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"platform-service/internal/app/principal"
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
	CompanyID    *string `json:"company_id"`
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

// companyToken menerbitkan token FAKE dengan claim cid sembarang untuk user nyata.
// Hanya untuk skenario yang tidak bisa dibuat lewat API (mis. cid company tempat user
// bukan member — switch-company akan menolaknya). Untuk alur normal pakai mustCompanyOwner.
func companyToken(userID, companyID string) map[string]string {
	return testhelper.BearerHeader(testSrv.Auth.Issue(principalUser(userID, companyID)))
}

// mustCreateCompany membuat company via API dengan token pemanggil dan mengembalikan ID-nya.
func mustCreateCompany(t *testing.T, headers map[string]string) string {
	t.Helper()
	slug := testhelper.UniqueCode("co")
	w := testSrv.POST("/api/v1/companies", map[string]any{"name": "Company " + slug, "slug": slug}, headers)
	require.Equal(t, 201, w.Code, "mustCreateCompany body=%s", w.Body.String())
	var got struct {
		ID string `json:"id"`
	}
	testhelper.DecodeBody(t, w, &got)
	return got.ID
}

// mustCompanyOwner menyiapkan user baru yang memiliki company siap pakai lewat alur nyata:
// login ke solution baru → buat company (otomatis member) → admin membuat plan & subscription
// → POST /auth/switch-company. Mengembalikan header token ber-cid, ID user, dan ID company.
func mustCompanyOwner(t *testing.T) (map[string]string, string, string) {
	t.Helper()
	tok, userID, solutionCode := mustLoginNewUser(t)
	companyID := mustCreateCompany(t, testhelper.BearerHeader(tok.AccessToken))
	solutionID := mustSolutionID(t, solutionCode)
	mustSubscribe(t, companyID, mustCreatePlan(t, solutionID, "basic", nil))
	switched := mustSwitchCompany(t, tok.AccessToken, companyID)
	return testhelper.BearerHeader(switched.AccessToken), userID, companyID
}

// mustSolutionID mencari ID solution berdasarkan code.
func mustSolutionID(t *testing.T, code string) string {
	t.Helper()
	var id string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(), `SELECT id FROM solutions WHERE code = $1`, code).Scan(&id))
	return id
}

// mustSubscribe membuat subscription company ke plan (admin) dan mengembalikan ID-nya.
func mustSubscribe(t *testing.T, companyID, planID string) string {
	t.Helper()
	w := testSrv.POST("/api/v1/companies/"+companyID+"/subscriptions", map[string]any{"plan_id": planID}, adminHeader())
	require.Equal(t, 201, w.Code, "mustSubscribe body=%s", w.Body.String())
	var got struct {
		ID string `json:"id"`
	}
	testhelper.DecodeBody(t, w, &got)
	return got.ID
}

// mustSwitchCompany memilih company aktif untuk sesi token tersebut.
func mustSwitchCompany(t *testing.T, accessToken, companyID string) tokenPair {
	t.Helper()
	w := testSrv.POST("/api/v1/auth/switch-company", map[string]any{"company_id": companyID}, testhelper.BearerHeader(accessToken))
	require.Equal(t, 200, w.Code, "mustSwitchCompany body=%s", w.Body.String())
	return decodeTokens(w.Body.Bytes())
}

func countEvents(t *testing.T, eventType, companyID string) int {
	t.Helper()
	var n int
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM events WHERE event_type = $1 AND company_id = $2`, eventType, companyID).Scan(&n))
	return n
}

// mustCreateBranch membuat branch di company via API dan mengembalikan ID-nya.
func mustCreateBranch(t *testing.T, headers map[string]string, companyID, code string) string {
	t.Helper()
	w := testSrv.POST("/api/v1/companies/"+companyID+"/branches", map[string]any{"name": "Branch " + code, "code": code}, headers)
	require.Equal(t, 201, w.Code, "mustCreateBranch body=%s", w.Body.String())
	var got struct {
		ID string `json:"id"`
	}
	testhelper.DecodeBody(t, w, &got)
	return got.ID
}

func principalUser(userID, companyID string) principal.Principal {
	return principal.Principal{Kind: principal.KindUser, UserID: userID, SessionID: uuid.NewString(), CompanyID: companyID}
}

// mustCreateBenefit membuat benefit via API (admin) dengan field minimal dan mengembalikan ID-nya.
// extra boleh berisi field tambahan (metering, usage_reset, unit, ...).
func mustCreateBenefit(t *testing.T, solutionID, code, valueType, aggregation string, extra map[string]any) string {
	t.Helper()
	body := map[string]any{"code": code, "name": "Benefit " + code, "value_type": valueType, "aggregation_strategy": aggregation}
	for k, v := range extra {
		body[k] = v
	}
	w := testSrv.POST("/api/v1/solutions/"+solutionID+"/benefits", body, adminHeader())
	require.Equal(t, 201, w.Code, "mustCreateBenefit body=%s", w.Body.String())
	var got struct {
		ID string `json:"id"`
	}
	testhelper.DecodeBody(t, w, &got)
	return got.ID
}

// mustCreatePlan membuat plan via API (admin) dan mengembalikan ID-nya. extra menimpa field default.
func mustCreatePlan(t *testing.T, solutionID, code string, extra map[string]any) string {
	t.Helper()
	body := map[string]any{"code": code, "name": "Plan " + code, "price": "100000"}
	for k, v := range extra {
		body[k] = v
	}
	w := testSrv.POST("/api/v1/solutions/"+solutionID+"/plans", body, adminHeader())
	require.Equal(t, 201, w.Code, "mustCreatePlan body=%s", w.Body.String())
	var got struct {
		ID string `json:"id"`
	}
	testhelper.DecodeBody(t, w, &got)
	return got.ID
}

// mustCreateAddon membuat add-on via API (admin) dan mengembalikan ID-nya. extra menimpa field default.
func mustCreateAddon(t *testing.T, solutionID, code string, extra map[string]any) string {
	t.Helper()
	body := map[string]any{"code": code, "name": "Add-on " + code, "price": "25000"}
	for k, v := range extra {
		body[k] = v
	}
	w := testSrv.POST("/api/v1/solutions/"+solutionID+"/addons", body, adminHeader())
	require.Equal(t, 201, w.Code, "mustCreateAddon body=%s", w.Body.String())
	var got struct {
		ID string `json:"id"`
	}
	testhelper.DecodeBody(t, w, &got)
	return got.ID
}

func countEventsBySolution(t *testing.T, eventType, solutionID string) int {
	t.Helper()
	var n int
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM events WHERE event_type = $1 AND solution_id = $2`, eventType, solutionID).Scan(&n))
	return n
}

type orgUnitData struct {
	ID        string `json:"id"`
	CompanyID string `json:"company_id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Status    string `json:"status"`
}

type orgUnitListResp struct {
	Data []orgUnitData `json:"data"`
	Meta struct {
		Total int64 `json:"total"`
	} `json:"meta"`
}

func orgCodes(items []orgUnitData) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Code)
	}
	return out
}
