package api_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/testhelper"
)

type solutionData struct {
	ID                       string  `json:"id"`
	Code                     string  `json:"code"`
	Name                     string  `json:"name"`
	Description              *string `json:"description"`
	Status                   string  `json:"status"`
	PermissionCatalogVersion *string `json:"permission_catalog_version"`
	CreatedAt                string  `json:"created_at"`
}

type solutionListResp struct {
	Data []solutionData `json:"data"`
	Meta struct {
		CurrentPage int64 `json:"current_page"`
		PerPage     int64 `json:"per_page"`
		Total       int64 `json:"total"`
		TotalPages  int64 `json:"total_pages"`
	} `json:"meta"`
}

func solutionCodes(items []solutionData) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Code)
	}
	return out
}

// ── Create ───────────────────────────────────────────────────────────────────

func TestSolution_Create_Unauthenticated(t *testing.T) {
	w := testSrv.POST("/api/v1/solutions", map[string]any{"code": testhelper.UniqueCode("sol"), "name": "X"})
	assert.Equal(t, 401, w.Code)
}

func TestSolution_Create_Forbidden_NonAdmin(t *testing.T) {
	w := testSrv.POST("/api/v1/solutions", map[string]any{"code": testhelper.UniqueCode("sol"), "name": "X"}, userHeader())
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestSolution_Create_Forbidden_ServiceToken(t *testing.T) {
	tok := testSrv.Auth.ServiceToken(uuid.NewString(), "catalog:sync")
	w := testSrv.POST("/api/v1/solutions", map[string]any{"code": testhelper.UniqueCode("sol"), "name": "X"}, testhelper.BearerHeader(tok))
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestSolution_Create_Forbidden_LicenseManaged(t *testing.T) {
	w := testOnPremSrv.POST("/api/v1/solutions", map[string]any{"code": testhelper.UniqueCode("sol"), "name": "X"}, adminHeader())
	require.Equal(t, 403, w.Code, w.Body.String())
	assert.Equal(t, "LICENSE_MANAGED", testhelper.DecodeError(t, w).ErrorCode)
}

func TestSolution_Create_InvalidPayload(t *testing.T) {
	w := testSrv.POST("/api/v1/solutions", map[string]any{"code": testhelper.UniqueCode("sol")}, adminHeader())
	require.Equal(t, 422, w.Code, w.Body.String())

	var fields map[string]string
	require.NoError(t, json.Unmarshal(testhelper.DecodeError(t, w).Errors, &fields))
	assert.Contains(t, fields, "name")
}

func TestSolution_Create_InvalidCode(t *testing.T) {
	w := testSrv.POST("/api/v1/solutions", map[string]any{"code": "Easy HR", "name": "Easy HR"}, adminHeader())
	require.Equal(t, 422, w.Code, w.Body.String())

	var msg string
	require.NoError(t, json.Unmarshal(testhelper.DecodeError(t, w).Errors, &msg))
	assert.Contains(t, msg, "lowercase")
}

func TestSolution_Create_InvalidStatus(t *testing.T) {
	w := testSrv.POST("/api/v1/solutions", map[string]any{"code": testhelper.UniqueCode("sol"), "name": "X", "status": "deleted"}, adminHeader())
	assert.Equal(t, 422, w.Code, w.Body.String())
}

func TestSolution_Create_Duplicate(t *testing.T) {
	code := testhelper.UniqueCode("dup")
	mustCreateSolution(t, code)

	w := testSrv.POST("/api/v1/solutions", map[string]any{"code": code, "name": "Lagi"}, adminHeader())
	require.Equal(t, 409, w.Code, w.Body.String())
	assert.Equal(t, "ERR_CONFLICT", testhelper.DecodeError(t, w).ErrorCode)
}

func TestSolution_Create_Success(t *testing.T) {
	code := testhelper.UniqueCode("easy-hr")
	w := testSrv.POST("/api/v1/solutions", map[string]any{
		"code":        code,
		"name":        "  Easy HR  ",
		"description": "Aplikasi HR",
	}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())

	var got solutionData
	testhelper.DecodeBody(t, w, &got)
	assert.NotEmpty(t, got.ID)
	assert.Equal(t, code, got.Code)
	assert.Equal(t, "Easy HR", got.Name)
	require.NotNil(t, got.Description)
	assert.Equal(t, "Aplikasi HR", *got.Description)
	assert.Equal(t, "active", got.Status)
	assert.Nil(t, got.PermissionCatalogVersion)
	assert.NotEmpty(t, got.CreatedAt)

	var dbCode, dbStatus string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT code, status FROM solutions WHERE id = $1`, got.ID).Scan(&dbCode, &dbStatus))
	assert.Equal(t, code, dbCode)
	assert.Equal(t, "active", dbStatus)
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestSolution_List_Unauthenticated(t *testing.T) {
	w := testSrv.GET("/api/v1/solutions")
	assert.Equal(t, 401, w.Code)
}

func TestSolution_List_InvalidStatusFilter(t *testing.T) {
	w := testSrv.GET("/api/v1/solutions?status=deleted", adminHeader())
	assert.Equal(t, 422, w.Code, w.Body.String())
}

func TestSolution_List_NonAdminSeesOnlyActive(t *testing.T) {
	prefix := testhelper.UniqueCode("vis")
	testhelper.MustSeedSolution(t, testSrv.DB, prefix+"-a", "active")
	testhelper.MustSeedSolution(t, testSrv.DB, prefix+"-b", "inactive")

	// status=inactive diabaikan untuk non-admin.
	w := testSrv.GET("/api/v1/solutions?status=inactive&search="+prefix, userHeader())
	require.Equal(t, 200, w.Code, w.Body.String())

	var resp solutionListResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, []string{prefix + "-a"}, solutionCodes(resp.Data))
}

func TestSolution_List_AdminSeesAllAndFilters(t *testing.T) {
	prefix := testhelper.UniqueCode("adm")
	testhelper.MustSeedSolution(t, testSrv.DB, prefix+"-a", "active")
	testhelper.MustSeedSolution(t, testSrv.DB, prefix+"-b", "inactive")

	w := testSrv.GET("/api/v1/solutions?search="+prefix, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var all solutionListResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&all))
	assert.ElementsMatch(t, []string{prefix + "-a", prefix + "-b"}, solutionCodes(all.Data))

	w = testSrv.GET("/api/v1/solutions?status=inactive&search="+prefix, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var inactive solutionListResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&inactive))
	assert.Equal(t, []string{prefix + "-b"}, solutionCodes(inactive.Data))
}

func TestSolution_List_PaginationAndSort(t *testing.T) {
	prefix := testhelper.UniqueCode("pg")
	for _, suffix := range []string{"c", "a", "b"} {
		testhelper.MustSeedSolution(t, testSrv.DB, prefix+"-"+suffix, "active")
	}

	w := testSrv.GET("/api/v1/solutions?search="+prefix+"&sort_by=code&sort_type=asc&page=1&limit=2", userHeader())
	require.Equal(t, 200, w.Code, w.Body.String())

	var resp solutionListResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, []string{prefix + "-a", prefix + "-b"}, solutionCodes(resp.Data))
	assert.Equal(t, int64(1), resp.Meta.CurrentPage)
	assert.Equal(t, int64(2), resp.Meta.PerPage)
	assert.Equal(t, int64(3), resp.Meta.Total)
	assert.Equal(t, int64(2), resp.Meta.TotalPages)
}

func TestSolution_List_ServiceToken(t *testing.T) {
	code := testhelper.UniqueCode("svc")
	testhelper.MustSeedSolution(t, testSrv.DB, code, "active")
	tok := testSrv.Auth.ServiceToken(uuid.NewString())

	w := testSrv.GET("/api/v1/solutions?search="+code, testhelper.BearerHeader(tok))
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp solutionListResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, []string{code}, solutionCodes(resp.Data))
}

// ── Get ──────────────────────────────────────────────────────────────────────

func TestSolution_Get_Unauthenticated(t *testing.T) {
	w := testSrv.GET("/api/v1/solutions/" + uuid.NewString())
	assert.Equal(t, 401, w.Code)
}

func TestSolution_Get_InvalidID(t *testing.T) {
	w := testSrv.GET("/api/v1/solutions/bukan-uuid", userHeader())
	assert.Equal(t, 422, w.Code, w.Body.String())
}

func TestSolution_Get_NotFound(t *testing.T) {
	w := testSrv.GET("/api/v1/solutions/"+uuid.NewString(), userHeader())
	require.Equal(t, 404, w.Code, w.Body.String())
	assert.Equal(t, "ERR_NOT_FOUND", testhelper.DecodeError(t, w).ErrorCode)
}

func TestSolution_Get_InactiveHiddenFromNonAdmin(t *testing.T) {
	id := testhelper.MustSeedSolution(t, testSrv.DB, testhelper.UniqueCode("hid"), "inactive")

	w := testSrv.GET("/api/v1/solutions/"+id, userHeader())
	assert.Equal(t, 404, w.Code, w.Body.String())

	w = testSrv.GET("/api/v1/solutions/"+id, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got solutionData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "inactive", got.Status)
}

func TestSolution_Get_Success(t *testing.T) {
	code := testhelper.UniqueCode("get")
	id := mustCreateSolution(t, code)

	w := testSrv.GET("/api/v1/solutions/"+id, userHeader())
	require.Equal(t, 200, w.Code, w.Body.String())

	var got solutionData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, code, got.Code)
	assert.Equal(t, "active", got.Status)
}

func TestSolution_Get_OnPremReadAllowed(t *testing.T) {
	id := mustCreateSolution(t, testhelper.UniqueCode("onprem"))

	w := testOnPremSrv.GET("/api/v1/solutions/"+id, userHeader())
	assert.Equal(t, 200, w.Code, w.Body.String())
}

// ── Update ───────────────────────────────────────────────────────────────────

func solutionPath(id string) string { return "/api/v1/solutions/" + id }

func TestSolution_Update_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.PATCH(solutionPath(uuid.NewString()), map[string]any{"name": "X"}).Code)
}

func TestSolution_Update_Forbidden(t *testing.T) {
	id := mustCreateSolution(t, testhelper.UniqueCode("upd"))
	tok, _, _ := mustLoginNewUser(t)
	assert.Equal(t, 403, testSrv.PATCH(solutionPath(id), map[string]any{"name": "X"}, testhelper.BearerHeader(tok.AccessToken)).Code)
	w := testOnPremSrv.PATCH(solutionPath(id), map[string]any{"name": "X"}, adminHeader())
	require.Equal(t, 403, w.Code)
	assert.Equal(t, "LICENSE_MANAGED", testhelper.DecodeError(t, w).ErrorCode)
}

func TestSolution_Update_InvalidPayload(t *testing.T) {
	id := mustCreateSolution(t, testhelper.UniqueCode("upd"))
	assert.Equal(t, 422, testSrv.PATCH(solutionPath(id), map[string]any{"status": "deleted"}, adminHeader()).Code)
	assert.Equal(t, 422, testSrv.PATCH(solutionPath(id), map[string]any{"name": " "}, adminHeader()).Code)
}

func TestSolution_Update_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.PATCH(solutionPath(uuid.NewString()), map[string]any{"name": "X"}, adminHeader()).Code)
}

func TestSolution_Update_OwnSessionSolution(t *testing.T) {
	var id string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(), `SELECT id FROM solutions WHERE code = $1`, testConsoleSolution).Scan(&id))
	w := testSrv.PATCH(solutionPath(id), map[string]any{"status": "inactive"}, adminHeader())
	assert.Equal(t, 422, w.Code, w.Body.String())
}

func TestSolution_Update_Success(t *testing.T) {
	code := testhelper.UniqueCode("upd")
	id := mustCreateSolution(t, code)

	w := testSrv.PATCH(solutionPath(id), map[string]any{"name": "Easy HR", "description": "HRIS", "status": "inactive"}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got solutionData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, code, got.Code, "kode tidak berubah")
	assert.Equal(t, "Easy HR", got.Name)
	require.NotNil(t, got.Description)
	assert.Equal(t, "HRIS", *got.Description)
	assert.Equal(t, "inactive", got.Status)
}

// ── Admin summary ────────────────────────────────────────────────────────────

func TestDashboard_AdminSummary(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET("/api/v1/admin/summary").Code)
	tok, _, _ := mustLoginNewUser(t)
	assert.Equal(t, 403, testSrv.GET("/api/v1/admin/summary", testhelper.BearerHeader(tok.AccessToken)).Code)

	w := testSrv.GET("/api/v1/admin/summary", adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got struct {
		Users struct {
			Active int `json:"active"`
			Admins int `json:"admins"`
		} `json:"users"`
		Solutions struct {
			Active int `json:"active"`
		} `json:"solutions"`
	}
	testhelper.DecodeBody(t, w, &got)
	assert.Positive(t, got.Users.Active)
	assert.Positive(t, got.Users.Admins)
	assert.Positive(t, got.Solutions.Active)
}

// ── App links (SSO) ──────────────────────────────────────────────────────────

type solutionLinksData struct {
	LaunchURL    *string  `json:"launch_url"`
	IconURL      *string  `json:"icon_url"`
	RedirectURIs []string `json:"redirect_uris"`
}

func TestSolution_Create_InvalidAppURL(t *testing.T) {
	for _, body := range []map[string]any{
		{"launch_url": "http://kasir.example.com/"}, // http hanya untuk localhost
		{"launch_url": "/relative"},
		{"redirect_uris": []string{"https://kasir.example.com/cb#frag"}},
		{"redirect_uris": []string{"javascript:alert(1)"}},
	} {
		body["code"] = testhelper.UniqueCode("links")
		body["name"] = "Links"
		w := testSrv.POST("/api/v1/solutions", body, adminHeader())
		assert.Equal(t, 422, w.Code, "body=%v resp=%s", body, w.Body.String())
	}
}

func TestSolution_Create_WithAppLinks(t *testing.T) {
	w := testSrv.POST("/api/v1/solutions", map[string]any{
		"code": testhelper.UniqueCode("links"), "name": "Links",
		"launch_url":    "http://localhost:4000/",
		"icon_url":      "https://cdn.example.com/kasir.svg",
		"redirect_uris": []string{"http://localhost:4000/auth/callback", "http://localhost:4000/auth/callback"},
	}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())
	var got solutionLinksData
	testhelper.DecodeBody(t, w, &got)
	require.NotNil(t, got.LaunchURL)
	assert.Equal(t, "http://localhost:4000/", *got.LaunchURL)
	require.NotNil(t, got.IconURL)
	assert.Equal(t, []string{"http://localhost:4000/auth/callback"}, got.RedirectURIs, "duplikat dibuang")
}

func TestSolution_Update_AppLinks(t *testing.T) {
	s := mustCreateSSOSolution(t)

	// Field lain tidak menyentuh tautan.
	w := testSrv.PATCH(solutionPath(s.ID), map[string]any{"name": "Renamed"}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got solutionLinksData
	testhelper.DecodeBody(t, w, &got)
	require.NotNil(t, got.LaunchURL)
	assert.Equal(t, []string{s.RedirectURI}, got.RedirectURIs)

	// "" / [] mengosongkan.
	w = testSrv.PATCH(solutionPath(s.ID), map[string]any{"launch_url": "", "redirect_uris": []string{}}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	got = solutionLinksData{}
	testhelper.DecodeBody(t, w, &got)
	assert.Nil(t, got.LaunchURL)
	assert.Empty(t, got.RedirectURIs)
	assert.NotNil(t, got.RedirectURIs, "redirect_uris selalu array")
}

func TestSolution_Update_PrivateHTTPRejectedByDefault(t *testing.T) {
	s := mustCreateSSOSolution(t)
	w := testSrv.PATCH(solutionPath(s.ID), map[string]any{
		"launch_url":    "http://192.168.1.29:3000/auth/sso",
		"redirect_uris": []string{"http://192.168.1.29:3000/auth/callback"},
	}, adminHeader())
	assert.Equal(t, 422, w.Code, w.Body.String())
}

func TestSolution_Update_PrivateHTTPAllowedWhenFlagOn(t *testing.T) {
	srv := testSrv.MustDerive(testhelper.WithAllowPrivateHTTPAppURLs())
	s := mustCreateSSOSolution(t)

	w := srv.PATCH(solutionPath(s.ID), map[string]any{
		"launch_url":    "http://192.168.1.29:3000/auth/sso",
		"redirect_uris": []string{"http://192.168.1.29:3000/auth/callback", "http://10.0.0.5:3000/auth/callback"},
	}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got solutionLinksData
	testhelper.DecodeBody(t, w, &got)
	require.NotNil(t, got.LaunchURL)
	assert.Equal(t, "http://192.168.1.29:3000/auth/sso", *got.LaunchURL)
	assert.Equal(t, []string{"http://192.168.1.29:3000/auth/callback", "http://10.0.0.5:3000/auth/callback"}, got.RedirectURIs)

	// Host publik via http tetap ditolak walau flag aktif.
	w = srv.PATCH(solutionPath(s.ID), map[string]any{"launch_url": "http://8.8.8.8/"}, adminHeader())
	assert.Equal(t, 422, w.Code, w.Body.String())
	w = srv.PATCH(solutionPath(s.ID), map[string]any{"launch_url": "http://kasir.example.com/"}, adminHeader())
	assert.Equal(t, 422, w.Code, w.Body.String())
}

// ── CORS dari tautan aplikasi ────────────────────────────────────────────────

// corsServer — engine dengan CORS_ALLOW_ORIGINS terbatas (bukan "*") di atas DB yang sama.
func corsServer() *testhelper.TestServer {
	return testSrv.MustDerive(testhelper.WithCORSOrigins("https://static.example.com"))
}

func preflight(srv *testhelper.TestServer, origin string) *httptest.ResponseRecorder {
	return srv.OPTIONS("/api/v1/auth/login", map[string]string{
		"Origin":                        origin,
		"Access-Control-Request-Method": "POST",
	})
}

// mustCreateSolutionWithLinks membuat solution (via srv) dengan launch_url & redirect URI di host unik.
func mustCreateSolutionWithLinks(t *testing.T, srv *testhelper.TestServer, status string) (id, launchOrigin, redirectOrigin string) {
	t.Helper()
	launchOrigin = "https://" + testhelper.UniqueCode("app") + ".example.com"
	redirectOrigin = "https://" + testhelper.UniqueCode("auth") + ".example.com:8443"
	w := srv.POST("/api/v1/solutions", map[string]any{
		"code": testhelper.UniqueCode("cors"), "name": "CORS", "status": status,
		"launch_url":    launchOrigin + "/home",
		"redirect_uris": []string{redirectOrigin + "/auth/callback"},
	}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	return resp.Data.ID, launchOrigin, redirectOrigin
}

func TestSolution_CORS_StaticOrigin(t *testing.T) {
	w := preflight(corsServer(), "https://static.example.com")
	assert.Equal(t, 204, w.Code)
	assert.Equal(t, "https://static.example.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestSolution_CORS_OriginFromAppLinks(t *testing.T) {
	srv := corsServer()
	_, launchOrigin, redirectOrigin := mustCreateSolutionWithLinks(t, srv, "active")

	for _, origin := range []string{launchOrigin, redirectOrigin} {
		w := preflight(srv, origin)
		assert.Equal(t, 204, w.Code, origin)
		assert.Equal(t, origin, w.Header().Get("Access-Control-Allow-Origin"), origin)
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	}
}

func TestSolution_CORS_UnknownOrigin(t *testing.T) {
	w := preflight(corsServer(), "https://"+testhelper.UniqueCode("evil")+".example.com")
	assert.Equal(t, 403, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestSolution_CORS_InactiveSolution(t *testing.T) {
	srv := corsServer()
	_, launchOrigin, _ := mustCreateSolutionWithLinks(t, srv, "inactive")

	assert.Equal(t, 403, preflight(srv, launchOrigin).Code)
}

func TestSolution_CORS_FollowsUpdate(t *testing.T) {
	srv := corsServer()
	id, launchOrigin, _ := mustCreateSolutionWithLinks(t, srv, "active")
	require.Equal(t, 204, preflight(srv, launchOrigin).Code)

	newOrigin := "https://" + testhelper.UniqueCode("moved") + ".example.com"
	w := srv.PATCH(solutionPath(id), map[string]any{"launch_url": newOrigin + "/"}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())

	assert.Equal(t, 204, preflight(srv, newOrigin).Code, "origin baru langsung berlaku")
	assert.Equal(t, 403, preflight(srv, launchOrigin).Code, "origin lama dicabut")

	w = srv.PATCH(solutionPath(id), map[string]any{"status": "inactive"}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, 403, preflight(srv, newOrigin).Code, "solution nonaktif dicabut")
}
