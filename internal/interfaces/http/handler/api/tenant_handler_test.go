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

type tenantData struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
}

type tenantListResp struct {
	Data []tenantData `json:"data"`
	Meta struct {
		Total int64 `json:"total"`
	} `json:"meta"`
}

func tenantIDs(items []tenantData) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.ID)
	}
	return out
}

// ── Create ───────────────────────────────────────────────────────────────────

func TestTenant_Create_Unauthenticated(t *testing.T) {
	w := testSrv.POST("/api/v1/tenants", map[string]any{"name": "Acme", "slug": testhelper.UniqueCode("acme")})
	assert.Equal(t, 401, w.Code)
}

func TestTenant_Create_Forbidden_ServiceToken(t *testing.T) {
	w := testSrv.POST("/api/v1/tenants", map[string]any{"name": "Acme", "slug": testhelper.UniqueCode("acme")},
		testhelper.BearerHeader(testSrv.Auth.ServiceToken(uuid.NewString(), "users:write")))
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestTenant_Create_InvalidPayload(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)
	h := testhelper.BearerHeader(tok.AccessToken)

	w := testSrv.POST("/api/v1/tenants", map[string]any{"name": "Acme"}, h)
	assert.Equal(t, 422, w.Code, "tanpa slug: %s", w.Body.String())

	w = testSrv.POST("/api/v1/tenants", map[string]any{"name": "Acme", "slug": "pt contoh!"}, h)
	assert.Equal(t, 422, w.Code, "slug tidak valid: %s", w.Body.String())
}

func TestTenant_Create_DuplicateSlug(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)
	h := testhelper.BearerHeader(tok.AccessToken)
	slug := testhelper.UniqueCode("dup")
	require.Equal(t, 201, testSrv.POST("/api/v1/tenants", map[string]any{"name": "A", "slug": slug}, h).Code)

	w := testSrv.POST("/api/v1/tenants", map[string]any{"name": "B", "slug": upper(slug)}, h)
	assert.Equal(t, 409, w.Code, w.Body.String())
}

func TestTenant_Create_Success_CreatorBecomesMember(t *testing.T) {
	tok, userID, _ := mustLoginNewUser(t)
	slug := testhelper.UniqueCode("acme")

	w := testSrv.POST("/api/v1/tenants", map[string]any{"name": " Acme Corp ", "slug": slug}, testhelper.BearerHeader(tok.AccessToken))
	require.Equal(t, 201, w.Code, w.Body.String())

	var got tenantData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "Acme Corp", got.Name)
	assert.Equal(t, slug, got.Slug)
	assert.Equal(t, "active", got.Status)

	var status string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT status FROM tenant_members WHERE tenant_id = $1 AND user_id = $2`, got.ID, userID).Scan(&status))
	assert.Equal(t, "active", status)
	assert.Equal(t, 1, countEvents(t, "tenant_member.added", got.ID))
}

func TestTenant_Create_ByPlatformAdmin_NoMembership(t *testing.T) {
	tenantID := mustCreateTenant(t, adminHeader())

	var n int
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM tenant_members WHERE tenant_id = $1`, tenantID).Scan(&n))
	assert.Equal(t, 0, n)
	assert.Equal(t, 0, countEvents(t, "tenant_member.added", tenantID))
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestTenant_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET("/api/v1/tenants").Code)
}

func TestTenant_List_InvalidStatus(t *testing.T) {
	assert.Equal(t, 422, testSrv.GET("/api/v1/tenants?status=deleted", adminHeader()).Code)
}

func TestTenant_List_UserSeesOnlyOwnTenants_WithoutActiveTenant(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)
	h := testhelper.BearerHeader(tok.AccessToken) // tid kosong — List adalah endpoint pemilihan tenant
	mine1 := mustCreateTenant(t, h)
	mine2 := mustCreateTenant(t, h)
	_, _, others := mustTenantOwner(t)

	w := testSrv.GET("/api/v1/tenants?limit=100", h)
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp tenantListResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.ElementsMatch(t, []string{mine1, mine2}, tenantIDs(resp.Data))
	assert.NotContains(t, tenantIDs(resp.Data), others)
	assert.Equal(t, int64(2), resp.Meta.Total)
}

func TestTenant_List_AdminSeesAll(t *testing.T) {
	_, _, tenantID := mustTenantOwner(t)
	var slug string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(), `SELECT slug FROM tenants WHERE id = $1`, tenantID).Scan(&slug))

	w := testSrv.GET("/api/v1/tenants?search="+slug, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp tenantListResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, []string{tenantID}, tenantIDs(resp.Data))
}

// ── Get ──────────────────────────────────────────────────────────────────────

func TestTenant_Get_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET("/api/v1/tenants/"+uuid.NewString()).Code)
}

func TestTenant_Get_InvalidID(t *testing.T) {
	assert.Equal(t, 422, testSrv.GET("/api/v1/tenants/bukan-uuid", adminHeader()).Code)
}

func TestTenant_Get_Forbidden_TenantNotSelected(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)
	h := testhelper.BearerHeader(tok.AccessToken)
	tenantID := mustCreateTenant(t, h)

	w := testSrv.GET("/api/v1/tenants/"+tenantID, h)
	require.Equal(t, 403, w.Code, w.Body.String())
	assert.Equal(t, "Select an active tenant first.", errorMessage(t, testhelper.DecodeError(t, w)))
}

func TestTenant_Get_Forbidden_OtherActiveTenant(t *testing.T) {
	_, userID, _ := mustTenantOwner(t)
	_, _, otherTenant := mustTenantOwner(t)

	w := testSrv.GET("/api/v1/tenants/"+otherTenant, tenantToken(userID, otherTenant))
	assert.Equal(t, 403, w.Code, "bukan member walau tid diisi: %s", w.Body.String())
}

func TestTenant_Get_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.GET("/api/v1/tenants/"+uuid.NewString(), adminHeader()).Code)
}

func TestTenant_Get_Success_Member(t *testing.T) {
	h, _, tenantID := mustTenantOwner(t)
	w := testSrv.GET("/api/v1/tenants/"+tenantID, h)
	require.Equal(t, 200, w.Code, w.Body.String())
	var got tenantData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, tenantID, got.ID)
}

func TestTenant_Get_Success_PlatformAdmin(t *testing.T) {
	_, _, tenantID := mustTenantOwner(t)
	assert.Equal(t, 200, testSrv.GET("/api/v1/tenants/"+tenantID, adminHeader()).Code)
}

func TestTenant_Get_Forbidden_InactiveMember(t *testing.T) {
	h, userID, tenantID := mustTenantOwner(t)
	_, err := testSrv.DB.ExecContext(context.Background(),
		`UPDATE tenant_members SET status = 'inactive' WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID)
	require.NoError(t, err)

	assert.Equal(t, 403, testSrv.GET("/api/v1/tenants/"+tenantID, h).Code)
}
