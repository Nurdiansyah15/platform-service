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

func branchesPath(companyID string) string {
	return "/api/v1/companies/" + companyID + "/branches"
}

// ── Create ───────────────────────────────────────────────────────────────────

func TestBranch_Create_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(branchesPath(uuid.NewString()), map[string]any{"name": "X", "code": "X"}).Code)
}

func TestBranch_Create_InvalidCompanyID(t *testing.T) {
	assert.Equal(t, 422, testSrv.POST(branchesPath("bukan-uuid"), map[string]any{"name": "X", "code": "X"}, adminHeader()).Code)
}

func TestBranch_Create_InvalidPayload(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	assert.Equal(t, 422, testSrv.POST(branchesPath(companyID), map[string]any{"code": "B1"}, h).Code)
}

func TestBranch_Create_CompanyNotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.POST(branchesPath(uuid.NewString()), map[string]any{"name": "X", "code": "X"}, adminHeader()).Code)
}

func TestBranch_Create_Forbidden_OtherCompany(t *testing.T) {
	_, _, companyID := mustCompanyOwner(t)
	strangerH, _, _ := mustCompanyOwner(t)

	w := testSrv.POST(branchesPath(companyID), map[string]any{"name": "X", "code": "X"}, strangerH)
	assert.Equal(t, 403, w.Code, "company company lain ditolak: %s", w.Body.String())
}

func TestBranch_Create_Forbidden_CompanyNotSelected(t *testing.T) {
	_, ownerID, companyID := mustCompanyOwner(t)
	noCompany := testhelper.BearerHeader(testSrv.Auth.Issue(principalUser(ownerID, "")))

	assert.Equal(t, 403, testSrv.POST(branchesPath(companyID), map[string]any{"name": "X", "code": "X"}, noCompany).Code)
}

func TestBranch_Create_Duplicate(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	require.Equal(t, 201, testSrv.POST(branchesPath(companyID), map[string]any{"name": "Cabang 1", "code": "b1"}, h).Code)

	assert.Equal(t, 409, testSrv.POST(branchesPath(companyID), map[string]any{"name": "Lagi", "code": "B1"}, h).Code)
}

func TestBranch_Create_Success(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)

	w := testSrv.POST(branchesPath(companyID), map[string]any{"name": "Cabang Bandung", "code": "bdg", "status": "inactive"}, h)
	require.Equal(t, 201, w.Code, w.Body.String())
	var got orgUnitData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, companyID, got.CompanyID)
	assert.Equal(t, "BDG", got.Code)
	assert.Equal(t, "inactive", got.Status)

	var eventCompany string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT company_id FROM events WHERE event_type = 'branch.created' AND payload->>'id' = $1`, got.ID).Scan(&eventCompany))
	assert.Equal(t, companyID, eventCompany, "event branch membawa company_id untuk routing webhook")
}

func TestBranch_Create_Success_PlatformAdmin(t *testing.T) {
	_, _, companyID := mustCompanyOwner(t)
	assert.Equal(t, 201, testSrv.POST(branchesPath(companyID), map[string]any{"name": "X", "code": "X"}, adminHeader()).Code)
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestBranch_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(branchesPath(uuid.NewString())).Code)
}

func TestBranch_List_Forbidden_OtherCompany(t *testing.T) {
	_, _, companyID := mustCompanyOwner(t)
	strangerH, _, _ := mustCompanyOwner(t)
	assert.Equal(t, 403, testSrv.GET(branchesPath(companyID), strangerH).Code)
}

func TestBranch_List_Success(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	for _, code := range []string{"B2", "B1"} {
		require.Equal(t, 201, testSrv.POST(branchesPath(companyID), map[string]any{"name": "Cabang " + code, "code": code}, h).Code)
	}
	// Branch di company lain tidak ikut.
	otherCompany := mustCreateCompany(t, adminHeader())
	mustCreateBranch(t, adminHeader(), otherCompany, "X1")

	w := testSrv.GET(branchesPath(companyID)+"?search=b", h)
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp orgUnitListResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, []string{"B1", "B2"}, orgCodes(resp.Data))
}
