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

func companiesPath(tenantID string) string {
	return "/api/v1/tenants/" + tenantID + "/companies"
}

type orgUnitData struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
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

// ── Create ───────────────────────────────────────────────────────────────────

func TestCompany_Create_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(companiesPath(uuid.NewString()), map[string]any{"name": "X", "code": "X"}).Code)
}

func TestCompany_Create_Forbidden_ServiceToken(t *testing.T) {
	w := testSrv.POST(companiesPath(uuid.NewString()), map[string]any{"name": "X", "code": "X"},
		testhelper.BearerHeader(testSrv.Auth.ServiceToken(uuid.NewString(), "org:write")))
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestCompany_Create_Forbidden_TenantNotSelected(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)
	h := testhelper.BearerHeader(tok.AccessToken)
	tenantID := mustCreateTenant(t, h)

	w := testSrv.POST(companiesPath(tenantID), map[string]any{"name": "HQ", "code": "HQ"}, h)
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestCompany_Create_Forbidden_NonMember(t *testing.T) {
	_, _, tenantID := mustTenantOwner(t)
	_, strangerID, _ := mustTenantOwner(t)
	w := testSrv.POST(companiesPath(tenantID), map[string]any{"name": "HQ", "code": "HQ"}, tenantToken(strangerID, tenantID))
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestCompany_Create_InvalidPayload(t *testing.T) {
	h, _, tenantID := mustTenantOwner(t)
	tests := map[string]map[string]any{
		"tanpa code":       {"name": "HQ"},
		"code ber-spasi":   {"name": "HQ", "code": "kantor pusat"},
		"status tak valid": {"name": "HQ", "code": "HQ", "status": "closed"},
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, 422, testSrv.POST(companiesPath(tenantID), body, h).Code)
		})
	}
}

func TestCompany_Create_TenantNotFound(t *testing.T) {
	w := testSrv.POST(companiesPath(uuid.NewString()), map[string]any{"name": "HQ", "code": "HQ"}, adminHeader())
	assert.Equal(t, 404, w.Code, w.Body.String())
}

func TestCompany_Create_DuplicateCodeCaseInsensitive(t *testing.T) {
	h, _, tenantID := mustTenantOwner(t)
	mustCreateCompany(t, h, tenantID, "hq-01")

	w := testSrv.POST(companiesPath(tenantID), map[string]any{"name": "Lagi", "code": "HQ-01"}, h)
	assert.Equal(t, 409, w.Code, w.Body.String())
}

func TestCompany_Create_SameCodeInOtherTenantAllowed(t *testing.T) {
	h1, _, t1 := mustTenantOwner(t)
	h2, _, t2 := mustTenantOwner(t)
	mustCreateCompany(t, h1, t1, "HQ")
	mustCreateCompany(t, h2, t2, "HQ")
}

func TestCompany_Create_CodeReusableAfterSoftDelete(t *testing.T) {
	h, _, tenantID := mustTenantOwner(t)
	id := mustCreateCompany(t, h, tenantID, "OLD")
	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE companies SET deleted_at = NOW() WHERE id = $1`, id)
	require.NoError(t, err)

	mustCreateCompany(t, h, tenantID, "OLD")
}

func TestCompany_Create_Success(t *testing.T) {
	h, _, tenantID := mustTenantOwner(t)

	w := testSrv.POST(companiesPath(tenantID), map[string]any{"name": " Kantor Pusat ", "code": " hq-01 "}, h)
	require.Equal(t, 201, w.Code, w.Body.String())
	var got orgUnitData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, tenantID, got.TenantID)
	assert.Equal(t, "Kantor Pusat", got.Name)
	assert.Equal(t, "HQ-01", got.Code)
	assert.Equal(t, "active", got.Status)

	var payloadCode string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT payload->>'code' FROM events WHERE event_type = 'company.created' AND payload->>'id' = $1`, got.ID).Scan(&payloadCode))
	assert.Equal(t, "HQ-01", payloadCode)
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestCompany_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(companiesPath(uuid.NewString())).Code)
}

func TestCompany_List_Forbidden_OtherTenant(t *testing.T) {
	_, _, tenantID := mustTenantOwner(t)
	_, strangerID, strangerTenant := mustTenantOwner(t)
	assert.Equal(t, 403, testSrv.GET(companiesPath(tenantID), tenantToken(strangerID, strangerTenant)).Code)
}

func TestCompany_List_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.GET(companiesPath(uuid.NewString()), adminHeader()).Code)
}

func TestCompany_List_Success_ExcludesDeleted(t *testing.T) {
	h, _, tenantID := mustTenantOwner(t)
	mustCreateCompany(t, h, tenantID, "B")
	mustCreateCompany(t, h, tenantID, "A")
	deleted := mustCreateCompany(t, h, tenantID, "C")
	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE companies SET deleted_at = NOW() WHERE id = $1`, deleted)
	require.NoError(t, err)

	w := testSrv.GET(companiesPath(tenantID), h)
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp orgUnitListResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, []string{"A", "B"}, orgCodes(resp.Data))
	assert.Equal(t, int64(2), resp.Meta.Total)
}
