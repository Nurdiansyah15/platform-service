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

// mustOwnedCompany: company dibuat user (otomatis member); subscribe=true → berlangganan solution user.
func (u userWithSolution) mustOwnedCompany(t *testing.T, subscribe bool) string {
	t.Helper()
	companyID := mustCreateCompany(t, testhelper.BearerHeader(u.tok.AccessToken))
	if subscribe {
		mustSubscribe(t, companyID, u.planID)
	}
	return companyID
}

func selectableIDs(t *testing.T, accessToken string) []string {
	t.Helper()
	w := testSrv.GET("/api/v1/auth/companies", testhelper.BearerHeader(accessToken))
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

// ── GET /auth/companies ────────────────────────────────────────────────────────

func TestAuthCompanies_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET("/api/v1/auth/companies").Code)
}

func TestAuthCompanies_OnlySelectable(t *testing.T) {
	u := newUserWithSolution(t)
	subscribed := u.mustOwnedCompany(t, true)
	u.mustOwnedCompany(t, false) // member tapi company tidak subscribe solution ini
	suspended := u.mustOwnedCompany(t, true)
	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE companies SET status = 'suspended' WHERE id = $1`, suspended)
	require.NoError(t, err)

	// Company lain yang subscribe tapi user bukan member.
	notMember := mustCreateCompany(t, adminHeader())
	mustSubscribe(t, notMember, u.planID)

	assert.Equal(t, []string{subscribed}, selectableIDs(t, u.tok.AccessToken))
}

// ── POST /auth/switch-company ─────────────────────────────────────────────────

func TestSwitchCompany_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST("/api/v1/auth/switch-company", map[string]any{"company_id": uuid.NewString()}).Code)
}

func TestSwitchCompany_InvalidPayload(t *testing.T) {
	u := newUserWithSolution(t)
	assert.Equal(t, 422, testSrv.POST("/api/v1/auth/switch-company", map[string]any{"company_id": "x"}, testhelper.BearerHeader(u.tok.AccessToken)).Code)
}

func TestSwitchCompany_Forbidden(t *testing.T) {
	u := newUserWithSolution(t)
	h := testhelper.BearerHeader(u.tok.AccessToken)

	notSubscribed := u.mustOwnedCompany(t, false)
	notMember := mustCreateCompany(t, adminHeader())
	mustSubscribe(t, notMember, u.planID)
	suspended := u.mustOwnedCompany(t, true)
	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE companies SET status = 'suspended' WHERE id = $1`, suspended)
	require.NoError(t, err)

	for name, companyID := range map[string]string{
		"tidak subscribe": notSubscribed, "bukan member": notMember, "company suspended": suspended, "tidak ada": uuid.NewString(),
	} {
		t.Run(name, func(t *testing.T) {
			w := testSrv.POST("/api/v1/auth/switch-company", map[string]any{"company_id": companyID}, h)
			assert.Equal(t, 403, w.Code, w.Body.String())
		})
	}
}

func TestSwitchCompany_Success(t *testing.T) {
	u := newUserWithSolution(t)
	companyID := u.mustOwnedCompany(t, true)

	switched := mustSwitchCompany(t, u.tok.AccessToken, companyID)
	require.NotNil(t, switched.CompanyID)
	assert.Equal(t, companyID, *switched.CompanyID)
	assert.Equal(t, u.tok.SessionID, switched.SessionID, "sesi yang sama")
	assert.Equal(t, companyID, jwtClaims(t, switched.AccessToken)["cid"])

	// Token baru bisa mengakses endpoint per-company.
	assert.Equal(t, 200, testSrv.GET("/api/v1/companies/"+companyID, testhelper.BearerHeader(switched.AccessToken)).Code)

	// Access token lama (cid kosong) ditolak platform karena cid tidak lagi cocok dengan sesi.
	assert.Equal(t, 401, testSrv.GET("/api/v1/auth/me", testhelper.BearerHeader(u.tok.AccessToken)).Code)
	// Refresh token lama ikut dirotasi.
	assert.Equal(t, 401, testSrv.POST("/api/v1/auth/refresh", map[string]any{"refresh_token": u.tok.RefreshToken}).Code)

	// Refresh dengan token baru mempertahankan company aktif.
	w := testSrv.POST("/api/v1/auth/refresh", map[string]any{"refresh_token": switched.RefreshToken})
	require.Equal(t, 200, w.Code, w.Body.String())
	refreshed := decodeTokens(w.Body.Bytes())
	require.NotNil(t, refreshed.CompanyID)
	assert.Equal(t, companyID, *refreshed.CompanyID)

	// /auth/companies menandai company aktif.
	w = testSrv.GET("/api/v1/auth/companies", testhelper.BearerHeader(refreshed.AccessToken))
	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"is_active":true`)
}

// ── Login memilih company otomatis ────────────────────────────────────────────

func TestLogin_AutoSelectsSingleCompany(t *testing.T) {
	u := newUserWithSolution(t)
	companyID := u.mustOwnedCompany(t, true)

	tok := mustLogin(t, u.email, u.code)
	require.NotNil(t, tok.CompanyID)
	assert.Equal(t, companyID, *tok.CompanyID)
	assert.Equal(t, 200, testSrv.GET("/api/v1/companies/"+companyID, testhelper.BearerHeader(tok.AccessToken)).Code)
}

func TestLogin_MultipleCompanies_NoAutoSelect(t *testing.T) {
	u := newUserWithSolution(t)
	u.mustOwnedCompany(t, true)
	u.mustOwnedCompany(t, true)

	tok := mustLogin(t, u.email, u.code)
	assert.Nil(t, tok.CompanyID)
}

// ── Refresh & company ─────────────────────────────────────────────────────────

func TestRefresh_RejectedAfterMembershipInactive(t *testing.T) {
	u := newUserWithSolution(t)
	companyID := u.mustOwnedCompany(t, true)
	switched := mustSwitchCompany(t, u.tok.AccessToken, companyID)

	_, err := testSrv.DB.ExecContext(context.Background(),
		`UPDATE company_members SET status = 'inactive' WHERE company_id = $1 AND user_id = $2`, companyID, u.userID)
	require.NoError(t, err)
	assert.Equal(t, 401, testSrv.POST("/api/v1/auth/refresh", map[string]any{"refresh_token": switched.RefreshToken}).Code)
}

func TestRefresh_RejectedAfterCompanySuspended(t *testing.T) {
	u := newUserWithSolution(t)
	companyID := u.mustOwnedCompany(t, true)
	switched := mustSwitchCompany(t, u.tok.AccessToken, companyID)

	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE companies SET status = 'suspended' WHERE id = $1`, companyID)
	require.NoError(t, err)
	assert.Equal(t, 401, testSrv.POST("/api/v1/auth/refresh", map[string]any{"refresh_token": switched.RefreshToken}).Code)
}
