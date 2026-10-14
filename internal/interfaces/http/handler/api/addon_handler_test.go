package api_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/testhelper"
)

func addonsPath(solutionID string) string     { return "/api/v1/solutions/" + solutionID + "/addons" }
func addonBenefitsPath(addonID string) string { return "/api/v1/addons/" + addonID + "/benefits" }

// ── Create ───────────────────────────────────────────────────────────────────

func TestAddon_Create_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(addonsPath(uuid.NewString()), map[string]any{"code": "payroll", "name": "Payroll"}).Code)
}

func TestAddon_Create_Forbidden_NonAdmin(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("ad"))
	assert.Equal(t, 403, testSrv.POST(addonsPath(solID), map[string]any{"code": "payroll", "name": "Payroll"}, userHeader()).Code)
}

func TestAddon_Create_Forbidden_LicenseManaged(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("ad"))
	w := testOnPremSrv.POST(addonsPath(solID), map[string]any{"code": "payroll", "name": "Payroll"}, adminHeader())
	require.Equal(t, 403, w.Code, w.Body.String())
	assert.Equal(t, "LICENSE_MANAGED", testhelper.DecodeError(t, w).ErrorCode)
}

func TestAddon_Create_InvalidPayload(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("ad"))
	tests := map[string]map[string]any{
		"tanpa name":        {"code": "payroll"},
		"code ber-spasi":    {"code": "pay roll", "name": "Payroll"},
		"harga negatif":     {"code": "payroll", "name": "Payroll", "price": "-5"},
		"billing tak valid": {"code": "payroll", "name": "Payroll", "billing_cycle": "daily"},
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, 422, testSrv.POST(addonsPath(solID), body, adminHeader()).Code)
		})
	}
}

func TestAddon_Create_SolutionNotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.POST(addonsPath(uuid.NewString()), map[string]any{"code": "payroll", "name": "P"}, adminHeader()).Code)
}

func TestAddon_Create_Duplicate(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("ad"))
	mustCreateAddon(t, solID, "payroll", nil)
	assert.Equal(t, 409, testSrv.POST(addonsPath(solID), map[string]any{"code": "PAYROLL", "name": "Lagi"}, adminHeader()).Code)

	// Code add-on boleh sama dengan code plan di solution yang sama (tabel berbeda).
	mustCreatePlan(t, solID, "payroll", nil)
}

func TestAddon_Create_Success(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("ad"))
	w := testSrv.POST(addonsPath(solID), map[string]any{"code": "payroll", "name": "Payroll", "price": 25000}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())

	var got planData // bentuk respons add-on identik dengan plan
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, solID, got.SolutionID)
	assert.Equal(t, "payroll", got.Code)
	assert.Equal(t, "25000.00", got.Price)
	assert.Equal(t, "IDR", got.Currency)
	assert.True(t, got.IsActive)
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestAddon_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(addonsPath(uuid.NewString())).Code)
}

func TestAddon_List_SolutionNotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.GET(addonsPath(uuid.NewString()), userHeader()).Code)
}

func TestAddon_List_NonAdminSeesOnlyActive(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("ad"))
	mustCreateAddon(t, solID, "payroll", map[string]any{"price": "50000"})
	mustCreateAddon(t, solID, "attendance", map[string]any{"price": "20000"})
	mustCreateAddon(t, solID, "old", map[string]any{"is_active": false})

	codes := func(h map[string]string) []string {
		w := testSrv.GET(addonsPath(solID), h)
		require.Equal(t, 200, w.Code, w.Body.String())
		var resp struct {
			Data []planData `json:"data"`
		}
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		out := []string{}
		for _, a := range resp.Data {
			out = append(out, a.Code)
		}
		return out
	}
	assert.Equal(t, []string{"attendance", "payroll"}, codes(userHeader()))
	assert.Equal(t, []string{"attendance", "old", "payroll"}, codes(adminHeader()), "old=25000 di antara 20000 dan 50000")
}

// ── Add-on benefits ──────────────────────────────────────────────────────────

func TestAddonBenefits_Replace_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.PUT(addonBenefitsPath(uuid.NewString()), map[string]any{"benefits": []any{}}).Code)
}

func TestAddonBenefits_Replace_Forbidden_NonAdmin(t *testing.T) {
	addonID := mustCreateAddon(t, mustCreateSolution(t, testhelper.UniqueCode("ad")), "payroll", nil)
	assert.Equal(t, 403, testSrv.PUT(addonBenefitsPath(addonID), map[string]any{"benefits": []any{}}, userHeader()).Code)
}

func TestAddonBenefits_Replace_Forbidden_LicenseManaged(t *testing.T) {
	addonID := mustCreateAddon(t, mustCreateSolution(t, testhelper.UniqueCode("ad")), "payroll", nil)
	assert.Equal(t, 403, testOnPremSrv.PUT(addonBenefitsPath(addonID), map[string]any{"benefits": []any{}}, adminHeader()).Code)
}

func TestAddonBenefits_Replace_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.PUT(addonBenefitsPath(uuid.NewString()), map[string]any{"benefits": []any{}}, adminHeader()).Code)
}

func TestAddonBenefits_Replace_InvalidPayload(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("ad"))
	addonID := mustCreateAddon(t, solID, "payroll", nil)
	payrollModule := mustCreateBenefit(t, solID, "payroll_module", "boolean", "any_true", nil)
	foreign := mustCreateBenefit(t, mustCreateSolution(t, testhelper.UniqueCode("other")), "payroll_module", "boolean", "any_true", nil)

	tests := map[string]any{
		"nilai salah tipe":      map[string]any{"benefits": []any{map[string]any{"benefit_id": payrollModule, "value": 1}}},
		"benefit solution lain": map[string]any{"benefits": []any{map[string]any{"benefit_id": foreign, "value": true}}},
		"tanpa benefits":        map[string]any{},
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, 422, testSrv.PUT(addonBenefitsPath(addonID), body, adminHeader()).Code)
		})
	}
}

func TestAddonBenefits_Replace_And_List_Success(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("ad"))
	addonID := mustCreateAddon(t, solID, "extra-employees", nil)
	maxEmp := mustCreateBenefit(t, solID, "max_employees", "integer", "sum", nil)

	w := testSrv.PUT(addonBenefitsPath(addonID), map[string]any{"benefits": []any{
		map[string]any{"benefit_id": maxEmp, "value": 50},
	}}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	got := decodeAssigned(t, w.Body.Bytes())
	require.Len(t, got, 1)
	assert.Equal(t, "max_employees", got[0].Code)

	w = testSrv.GET(addonBenefitsPath(addonID), userHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	got = decodeAssigned(t, w.Body.Bytes())
	require.Len(t, got, 1)
	assert.JSONEq(t, `50`, string(got[0].Value))
}

func TestAddonBenefits_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(addonBenefitsPath(uuid.NewString())).Code)
}

func TestAddonBenefits_List_InactiveAddonHiddenFromNonAdmin(t *testing.T) {
	addonID := mustCreateAddon(t, mustCreateSolution(t, testhelper.UniqueCode("ad")), "old", map[string]any{"is_active": false})
	assert.Equal(t, 404, testSrv.GET(addonBenefitsPath(addonID), userHeader()).Code)
	assert.Equal(t, 200, testSrv.GET(addonBenefitsPath(addonID), adminHeader()).Code)
}
