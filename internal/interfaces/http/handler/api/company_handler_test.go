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

type companyData struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
}

type companyListResp struct {
	Data []companyData `json:"data"`
	Meta struct {
		Total int64 `json:"total"`
	} `json:"meta"`
}

func companyIDs(items []companyData) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.ID)
	}
	return out
}

// ── Create ───────────────────────────────────────────────────────────────────

func TestCompany_Create_Unauthenticated(t *testing.T) {
	w := testSrv.POST("/api/v1/companies", map[string]any{"name": "Acme", "slug": testhelper.UniqueCode("acme")})
	assert.Equal(t, 401, w.Code)
}

func TestCompany_Create_Forbidden_ServiceToken(t *testing.T) {
	w := testSrv.POST("/api/v1/companies", map[string]any{"name": "Acme", "slug": testhelper.UniqueCode("acme")},
		testhelper.BearerHeader(testSrv.Auth.ServiceToken(uuid.NewString(), "users:write")))
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestCompany_Create_InvalidPayload(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)
	h := testhelper.BearerHeader(tok.AccessToken)

	w := testSrv.POST("/api/v1/companies", map[string]any{"name": "Acme"}, h)
	assert.Equal(t, 422, w.Code, "tanpa slug: %s", w.Body.String())

	w = testSrv.POST("/api/v1/companies", map[string]any{"name": "Acme", "slug": "pt contoh!"}, h)
	assert.Equal(t, 422, w.Code, "slug tidak valid: %s", w.Body.String())
}

func TestCompany_Create_DuplicateSlug(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)
	h := testhelper.BearerHeader(tok.AccessToken)
	slug := testhelper.UniqueCode("dup")
	require.Equal(t, 201, testSrv.POST("/api/v1/companies", map[string]any{"name": "A", "slug": slug}, h).Code)

	w := testSrv.POST("/api/v1/companies", map[string]any{"name": "B", "slug": upper(slug)}, h)
	assert.Equal(t, 409, w.Code, w.Body.String())
}

func TestCompany_Create_Success_CreatorBecomesMember(t *testing.T) {
	tok, userID, _ := mustLoginNewUser(t)
	slug := testhelper.UniqueCode("acme")

	w := testSrv.POST("/api/v1/companies", map[string]any{"name": " Acme Corp ", "slug": slug}, testhelper.BearerHeader(tok.AccessToken))
	require.Equal(t, 201, w.Code, w.Body.String())

	var got companyData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "Acme Corp", got.Name)
	assert.Equal(t, slug, got.Slug)
	assert.Equal(t, "active", got.Status)

	var status string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT status FROM company_members WHERE company_id = $1 AND user_id = $2`, got.ID, userID).Scan(&status))
	assert.Equal(t, "active", status)
	assert.Equal(t, 1, countEvents(t, "company_member.added", got.ID))
}

func TestCompany_Create_ByPlatformAdmin_NoMembership(t *testing.T) {
	companyID := mustCreateCompany(t, adminHeader())

	var n int
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM company_members WHERE company_id = $1`, companyID).Scan(&n))
	assert.Equal(t, 0, n)
	assert.Equal(t, 0, countEvents(t, "company_member.added", companyID))
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestCompany_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET("/api/v1/companies").Code)
}

func TestCompany_List_InvalidStatus(t *testing.T) {
	assert.Equal(t, 422, testSrv.GET("/api/v1/companies?status=deleted", adminHeader()).Code)
}

func TestCompany_List_UserSeesOnlyOwnCompanies_WithoutActiveCompany(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)
	h := testhelper.BearerHeader(tok.AccessToken) // cid kosong — List adalah endpoint pemilihan company
	mine1 := mustCreateCompany(t, h)
	mine2 := mustCreateCompany(t, h)
	_, _, others := mustCompanyOwner(t)

	w := testSrv.GET("/api/v1/companies?limit=100", h)
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp companyListResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.ElementsMatch(t, []string{mine1, mine2}, companyIDs(resp.Data))
	assert.NotContains(t, companyIDs(resp.Data), others)
	assert.Equal(t, int64(2), resp.Meta.Total)
}

func TestCompany_List_AdminSeesAll(t *testing.T) {
	_, _, companyID := mustCompanyOwner(t)
	var slug string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(), `SELECT slug FROM companies WHERE id = $1`, companyID).Scan(&slug))

	w := testSrv.GET("/api/v1/companies?search="+slug, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp companyListResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, []string{companyID}, companyIDs(resp.Data))
}

// ── Get ──────────────────────────────────────────────────────────────────────

func TestCompany_Get_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET("/api/v1/companies/"+uuid.NewString()).Code)
}

func TestCompany_Get_InvalidID(t *testing.T) {
	assert.Equal(t, 422, testSrv.GET("/api/v1/companies/bukan-uuid", adminHeader()).Code)
}

func TestCompany_Get_Forbidden_CompanyNotSelected(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)
	h := testhelper.BearerHeader(tok.AccessToken)
	companyID := mustCreateCompany(t, h)

	w := testSrv.GET("/api/v1/companies/"+companyID, h)
	require.Equal(t, 403, w.Code, w.Body.String())
	assert.Equal(t, "Select an active company first.", errorMessage(t, testhelper.DecodeError(t, w)))
}

func TestCompany_Get_Forbidden_OtherActiveCompany(t *testing.T) {
	_, userID, _ := mustCompanyOwner(t)
	_, _, otherCompany := mustCompanyOwner(t)

	w := testSrv.GET("/api/v1/companies/"+otherCompany, companyToken(userID, otherCompany))
	assert.Equal(t, 403, w.Code, "bukan member walau tid diisi: %s", w.Body.String())
}

func TestCompany_Get_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.GET("/api/v1/companies/"+uuid.NewString(), adminHeader()).Code)
}

func TestCompany_Get_Success_Member(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	w := testSrv.GET("/api/v1/companies/"+companyID, h)
	require.Equal(t, 200, w.Code, w.Body.String())
	var got companyData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, companyID, got.ID)
}

func TestCompany_Get_Success_PlatformAdmin(t *testing.T) {
	_, _, companyID := mustCompanyOwner(t)
	assert.Equal(t, 200, testSrv.GET("/api/v1/companies/"+companyID, adminHeader()).Code)
}

func TestCompany_Get_Forbidden_InactiveMember(t *testing.T) {
	h, userID, companyID := mustCompanyOwner(t)
	_, err := testSrv.DB.ExecContext(context.Background(),
		`UPDATE company_members SET status = 'inactive' WHERE company_id = $1 AND user_id = $2`, companyID, userID)
	require.NoError(t, err)

	assert.Equal(t, 403, testSrv.GET("/api/v1/companies/"+companyID, h).Code)
}

// ── Update ───────────────────────────────────────────────────────────────────

func companyPath(companyID string) string {
	return "/api/v1/companies/" + companyID
}

func TestCompany_Update_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.PATCH(companyPath(uuid.NewString()), map[string]any{"name": "X"}).Code)
}

func TestCompany_Update_Forbidden_NonMember(t *testing.T) {
	_, _, companyID := mustCompanyOwner(t)
	_, strangerID, _ := mustCompanyOwner(t)
	w := testSrv.PATCH(companyPath(companyID), map[string]any{"name": "X"}, companyToken(strangerID, companyID))
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestCompany_Update_Forbidden_StatusByMember(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	w := testSrv.PATCH(companyPath(companyID), map[string]any{"status": "suspended"}, h)
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestCompany_Update_InvalidPayload(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	for name, body := range map[string]map[string]any{
		"status tak dikenal": {"status": "deleted"},
		"name kosong":        {"name": "   "},
		"slug tidak valid":   {"slug": "Bukan Slug!"},
	} {
		t.Run(name, func(t *testing.T) {
			w := testSrv.PATCH(companyPath(companyID), body, h)
			assert.Equal(t, 422, w.Code, w.Body.String())
		})
	}
	assert.Equal(t, 422, testSrv.PATCH(companyPath("bukan-uuid"), map[string]any{"name": "X"}, adminHeader()).Code)
}

func TestCompany_Update_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.PATCH(companyPath(uuid.NewString()), map[string]any{"name": "X"}, adminHeader()).Code)
}

func TestCompany_Update_Duplicate(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	other := mustCreateCompany(t, adminHeader())
	var otherSlug string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(), `SELECT slug FROM companies WHERE id = $1`, other).Scan(&otherSlug))

	w := testSrv.PATCH(companyPath(companyID), map[string]any{"slug": otherSlug}, h)
	assert.Equal(t, 409, w.Code, w.Body.String())
}

func TestCompany_Update_Success(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	slug := testhelper.UniqueCode("renamed")

	w := testSrv.PATCH(companyPath(companyID), map[string]any{"name": "  Acme Baru  ", "slug": slug}, h)
	require.Equal(t, 200, w.Code, w.Body.String())
	var got companyData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "Acme Baru", got.Name)
	assert.Equal(t, slug, got.Slug)
	assert.Equal(t, "active", got.Status)
	assert.Equal(t, 1, countEvents(t, "company.updated", companyID))
}

func TestCompany_Update_Success_AdminChangesStatus(t *testing.T) {
	_, _, companyID := mustCompanyOwner(t)

	w := testSrv.PATCH(companyPath(companyID), map[string]any{"status": "suspended"}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got companyData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "suspended", got.Status)

	var status string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(), `SELECT status FROM companies WHERE id = $1`, companyID).Scan(&status))
	assert.Equal(t, "suspended", status)
}
