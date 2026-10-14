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

// userWithSolution: user baru yang login ke solution baru + ID solution & plan basic-nya.
type userWithSolution struct {
	tok        tokenPair
	email      string
	userID     string
	solutionID string
	code       string
	planID     string
}

func newUserWithSolution(t *testing.T) userWithSolution {
	t.Helper()
	u := userWithSolution{email: testhelper.UniqueEmail("sel"), code: testhelper.UniqueCode("app")}
	u.userID = mustCreateUser(t, u.email)
	u.solutionID = mustCreateSolution(t, u.code)
	u.planID = mustCreatePlan(t, u.solutionID, "basic", nil)
	u.tok = mustLogin(t, u.email, u.code)
	return u
}

// mustOwnedTenant: tenant dibuat user (otomatis member); subscribe=true → berlangganan solution user.
func (u userWithSolution) mustOwnedTenant(t *testing.T, subscribe bool) string {
	t.Helper()
	tenantID := mustCreateTenant(t, testhelper.BearerHeader(u.tok.AccessToken))
	if subscribe {
		mustSubscribe(t, tenantID, u.planID)
	}
	return tenantID
}

func selectableIDs(t *testing.T, accessToken string) []string {
	t.Helper()
	w := testSrv.GET("/api/v1/auth/tenants", testhelper.BearerHeader(accessToken))
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	out := []string{}
	for _, it := range resp.Data {
		out = append(out, it.ID)
	}
	return out
}

// ── GET /auth/tenants ────────────────────────────────────────────────────────

func TestAuthTenants_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET("/api/v1/auth/tenants").Code)
}

func TestAuthTenants_OnlySelectable(t *testing.T) {
	u := newUserWithSolution(t)
	subscribed := u.mustOwnedTenant(t, true)
	u.mustOwnedTenant(t, false) // member tapi tenant tidak subscribe solution ini
	suspended := u.mustOwnedTenant(t, true)
	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE tenants SET status = 'suspended' WHERE id = $1`, suspended)
	require.NoError(t, err)

	// Tenant lain yang subscribe tapi user bukan member.
	notMember := mustCreateTenant(t, adminHeader())
	mustSubscribe(t, notMember, u.planID)

	assert.Equal(t, []string{subscribed}, selectableIDs(t, u.tok.AccessToken))
}

// ── POST /auth/switch-tenant ─────────────────────────────────────────────────

func TestSwitchTenant_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST("/api/v1/auth/switch-tenant", map[string]any{"tenant_id": uuid.NewString()}).Code)
}

func TestSwitchTenant_InvalidPayload(t *testing.T) {
	u := newUserWithSolution(t)
	assert.Equal(t, 422, testSrv.POST("/api/v1/auth/switch-tenant", map[string]any{"tenant_id": "x"}, testhelper.BearerHeader(u.tok.AccessToken)).Code)
}

func TestSwitchTenant_Forbidden(t *testing.T) {
	u := newUserWithSolution(t)
	h := testhelper.BearerHeader(u.tok.AccessToken)

	notSubscribed := u.mustOwnedTenant(t, false)
	notMember := mustCreateTenant(t, adminHeader())
	mustSubscribe(t, notMember, u.planID)
	suspended := u.mustOwnedTenant(t, true)
	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE tenants SET status = 'suspended' WHERE id = $1`, suspended)
	require.NoError(t, err)

	for name, tenantID := range map[string]string{
		"tidak subscribe": notSubscribed, "bukan member": notMember, "tenant suspended": suspended, "tidak ada": uuid.NewString(),
	} {
		t.Run(name, func(t *testing.T) {
			w := testSrv.POST("/api/v1/auth/switch-tenant", map[string]any{"tenant_id": tenantID}, h)
			assert.Equal(t, 403, w.Code, w.Body.String())
		})
	}
}

func TestSwitchTenant_Success(t *testing.T) {
	u := newUserWithSolution(t)
	tenantID := u.mustOwnedTenant(t, true)

	switched := mustSwitchTenant(t, u.tok.AccessToken, tenantID)
	require.NotNil(t, switched.TenantID)
	assert.Equal(t, tenantID, *switched.TenantID)
	assert.Equal(t, u.tok.SessionID, switched.SessionID, "sesi yang sama")
	assert.Equal(t, tenantID, jwtClaims(t, switched.AccessToken)["tid"])

	// Token baru bisa mengakses endpoint per-tenant.
	assert.Equal(t, 200, testSrv.GET("/api/v1/tenants/"+tenantID, testhelper.BearerHeader(switched.AccessToken)).Code)

	// Access token lama (tid kosong) ditolak platform karena tid tidak lagi cocok dengan sesi.
	assert.Equal(t, 401, testSrv.GET("/api/v1/auth/me", testhelper.BearerHeader(u.tok.AccessToken)).Code)
	// Refresh token lama ikut dirotasi.
	assert.Equal(t, 401, testSrv.POST("/api/v1/auth/refresh", map[string]any{"refresh_token": u.tok.RefreshToken}).Code)

	// Refresh dengan token baru mempertahankan tenant aktif.
	w := testSrv.POST("/api/v1/auth/refresh", map[string]any{"refresh_token": switched.RefreshToken})
	require.Equal(t, 200, w.Code, w.Body.String())
	refreshed := decodeTokens(w.Body.Bytes())
	require.NotNil(t, refreshed.TenantID)
	assert.Equal(t, tenantID, *refreshed.TenantID)

	// /auth/tenants menandai tenant aktif.
	w = testSrv.GET("/api/v1/auth/tenants", testhelper.BearerHeader(refreshed.AccessToken))
	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"is_active":true`)
}

// ── Login memilih tenant otomatis ────────────────────────────────────────────

func TestLogin_AutoSelectsSingleTenant(t *testing.T) {
	u := newUserWithSolution(t)
	tenantID := u.mustOwnedTenant(t, true)

	tok := mustLogin(t, u.email, u.code)
	require.NotNil(t, tok.TenantID)
	assert.Equal(t, tenantID, *tok.TenantID)
	assert.Equal(t, 200, testSrv.GET("/api/v1/tenants/"+tenantID, testhelper.BearerHeader(tok.AccessToken)).Code)
}

func TestLogin_MultipleTenants_NoAutoSelect(t *testing.T) {
	u := newUserWithSolution(t)
	u.mustOwnedTenant(t, true)
	u.mustOwnedTenant(t, true)

	tok := mustLogin(t, u.email, u.code)
	assert.Nil(t, tok.TenantID)
}

// ── Refresh & tenant ─────────────────────────────────────────────────────────

func TestRefresh_RejectedAfterMembershipInactive(t *testing.T) {
	u := newUserWithSolution(t)
	tenantID := u.mustOwnedTenant(t, true)
	switched := mustSwitchTenant(t, u.tok.AccessToken, tenantID)

	_, err := testSrv.DB.ExecContext(context.Background(),
		`UPDATE tenant_members SET status = 'inactive' WHERE tenant_id = $1 AND user_id = $2`, tenantID, u.userID)
	require.NoError(t, err)
	assert.Equal(t, 401, testSrv.POST("/api/v1/auth/refresh", map[string]any{"refresh_token": switched.RefreshToken}).Code)
}

func TestRefresh_RejectedAfterTenantSuspended(t *testing.T) {
	u := newUserWithSolution(t)
	tenantID := u.mustOwnedTenant(t, true)
	switched := mustSwitchTenant(t, u.tok.AccessToken, tenantID)

	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE tenants SET status = 'suspended' WHERE id = $1`, tenantID)
	require.NoError(t, err)
	assert.Equal(t, 401, testSrv.POST("/api/v1/auth/refresh", map[string]any{"refresh_token": switched.RefreshToken}).Code)
}
