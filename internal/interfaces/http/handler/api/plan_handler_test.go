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

func plansPath(solutionID string) string    { return "/api/v1/solutions/" + solutionID + "/plans" }
func planBenefitsPath(planID string) string { return "/api/v1/plans/" + planID + "/benefits" }

type planData struct {
	ID           string `json:"id"`
	SolutionID   string `json:"solution_id"`
	Code         string `json:"code"`
	Price        string `json:"price"`
	Currency     string `json:"currency"`
	BillingCycle string `json:"billing_cycle"`
	IsActive     bool   `json:"is_active"`
}

type assignedBenefit struct {
	BenefitID string          `json:"benefit_id"`
	Code      string          `json:"code"`
	ValueType string          `json:"value_type"`
	Value     json.RawMessage `json:"value"`
}

func decodeAssigned(t *testing.T, body []byte) []assignedBenefit {
	t.Helper()
	var resp struct {
		Data []assignedBenefit `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &resp), string(body))
	return resp.Data
}

// catalogFixture: solution + plan + dua benefit (integer & boolean).
func catalogFixture(t *testing.T) (solutionID, planID, maxEmpID, payrollID string) {
	t.Helper()
	solutionID = mustCreateSolution(t, testhelper.UniqueCode("cat"))
	planID = mustCreatePlan(t, solutionID, "pro", nil)
	maxEmpID = mustCreateBenefit(t, solutionID, "max_employees", "integer", "sum", nil)
	payrollID = mustCreateBenefit(t, solutionID, "payroll_module", "boolean", "any_true", nil)
	return
}

// ── Create ───────────────────────────────────────────────────────────────────

func TestPlan_Create_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(plansPath(uuid.NewString()), map[string]any{"code": "pro", "name": "Pro"}).Code)
}

func TestPlan_Create_Forbidden_NonAdmin(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("pl"))
	assert.Equal(t, 403, testSrv.POST(plansPath(solID), map[string]any{"code": "pro", "name": "Pro"}, userHeader()).Code)
}

func TestPlan_Create_Forbidden_LicenseManaged(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("pl"))
	w := testOnPremSrv.POST(plansPath(solID), map[string]any{"code": "pro", "name": "Pro"}, adminHeader())
	require.Equal(t, 403, w.Code, w.Body.String())
	assert.Equal(t, "LICENSE_MANAGED", testhelper.DecodeError(t, w).ErrorCode)
}

func TestPlan_Create_InvalidPayload(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("pl"))
	tests := map[string]map[string]any{
		"tanpa name":        {"code": "pro"},
		"code ber-spasi":    {"code": "pro plan", "name": "Pro"},
		"harga negatif":     {"code": "pro", "name": "Pro", "price": -1},
		"harga 3 desimal":   {"code": "pro", "name": "Pro", "price": "1.234"},
		"harga bukan angka": {"code": "pro", "name": "Pro", "price": "gratis"},
		"harga notasi e":    {"code": "pro", "name": "Pro", "price": json.RawMessage(`1e5`)},
		"billing tak valid": {"code": "pro", "name": "Pro", "billing_cycle": "weekly"},
		"currency 2 huruf":  {"code": "pro", "name": "Pro", "currency": "RP"},
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, 422, testSrv.POST(plansPath(solID), body, adminHeader()).Code)
		})
	}
}

func TestPlan_Create_SolutionNotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.POST(plansPath(uuid.NewString()), map[string]any{"code": "pro", "name": "Pro"}, adminHeader()).Code)
}

func TestPlan_Create_Duplicate(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("pl"))
	mustCreatePlan(t, solID, "pro", nil)
	assert.Equal(t, 409, testSrv.POST(plansPath(solID), map[string]any{"code": "PRO", "name": "Lagi"}, adminHeader()).Code)
}

func TestPlan_Create_Success(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("pl"))
	w := testSrv.POST(plansPath(solID), map[string]any{
		"code": "Pro", "name": "Pro", "price": 150000.5, "currency": "idr", "billing_cycle": "yearly",
	}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())

	var got planData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, solID, got.SolutionID)
	assert.Equal(t, "pro", got.Code)
	assert.Equal(t, "150000.50", got.Price)
	assert.Equal(t, "IDR", got.Currency)
	assert.Equal(t, "yearly", got.BillingCycle)
	assert.True(t, got.IsActive)

	var dbPrice string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(), `SELECT price::text FROM plans WHERE id = $1`, got.ID).Scan(&dbPrice))
	assert.Equal(t, "150000.50", dbPrice)
}

func TestPlan_Create_Success_Defaults(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("pl"))
	w := testSrv.POST(plansPath(solID), map[string]any{"code": "free", "name": "Free"}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())
	var got planData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "0.00", got.Price)
	assert.Equal(t, "IDR", got.Currency)
	assert.Equal(t, "monthly", got.BillingCycle)
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestPlan_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(plansPath(uuid.NewString())).Code)
}

func TestPlan_List_SolutionNotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.GET(plansPath(uuid.NewString()), userHeader()).Code)
}

func TestPlan_List_NonAdminSeesOnlyActive_SortedByPrice(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("pl"))
	mustCreatePlan(t, solID, "pro", map[string]any{"price": "300000"})
	mustCreatePlan(t, solID, "basic", map[string]any{"price": "100000"})
	mustCreatePlan(t, solID, "legacy", map[string]any{"price": "50000", "is_active": false})

	codes := func(h map[string]string, query string) []string {
		w := testSrv.GET(plansPath(solID)+query, h)
		require.Equal(t, 200, w.Code, w.Body.String())
		var resp struct {
			Data []planData `json:"data"`
		}
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		out := []string{}
		for _, p := range resp.Data {
			out = append(out, p.Code)
		}
		return out
	}

	assert.Equal(t, []string{"basic", "pro"}, codes(userHeader(), "?is_active=false"), "filter is_active diabaikan untuk non-admin")
	assert.Equal(t, []string{"legacy", "basic", "pro"}, codes(adminHeader(), ""))
	assert.Equal(t, []string{"legacy"}, codes(adminHeader(), "?is_active=false"))
}

// ── Plan benefits ────────────────────────────────────────────────────────────

func TestPlanBenefits_Replace_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.PUT(planBenefitsPath(uuid.NewString()), map[string]any{"benefits": []any{}}).Code)
}

func TestPlanBenefits_Replace_Forbidden_NonAdmin(t *testing.T) {
	_, planID, _, _ := catalogFixture(t)
	assert.Equal(t, 403, testSrv.PUT(planBenefitsPath(planID), map[string]any{"benefits": []any{}}, userHeader()).Code)
}

func TestPlanBenefits_Replace_Forbidden_LicenseManaged(t *testing.T) {
	_, planID, _, _ := catalogFixture(t)
	w := testOnPremSrv.PUT(planBenefitsPath(planID), map[string]any{"benefits": []any{}}, adminHeader())
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestPlanBenefits_Replace_PlanNotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.PUT(planBenefitsPath(uuid.NewString()), map[string]any{"benefits": []any{}}, adminHeader()).Code)
}

func TestPlanBenefits_Replace_InvalidPayload(t *testing.T) {
	_, planID, maxEmpID, payrollID := catalogFixture(t)
	otherSolution := mustCreateSolution(t, testhelper.UniqueCode("other"))
	foreignBenefit := mustCreateBenefit(t, otherSolution, "max_employees", "integer", "sum", nil)

	tests := map[string]any{
		"tanpa field benefits":  map[string]any{},
		"benefit_id bukan uuid": map[string]any{"benefits": []any{map[string]any{"benefit_id": "x", "value": 1}}},
		"tanpa value":           map[string]any{"benefits": []any{map[string]any{"benefit_id": maxEmpID}}},
		"integer berdesimal":    map[string]any{"benefits": []any{map[string]any{"benefit_id": maxEmpID, "value": 1.5}}},
		"boolean berupa string": map[string]any{"benefits": []any{map[string]any{"benefit_id": payrollID, "value": "true"}}},
		"benefit duplikat": map[string]any{"benefits": []any{
			map[string]any{"benefit_id": maxEmpID, "value": 1}, map[string]any{"benefit_id": maxEmpID, "value": 2},
		}},
		"benefit tak dikenal":   map[string]any{"benefits": []any{map[string]any{"benefit_id": uuid.NewString(), "value": 1}}},
		"benefit solution lain": map[string]any{"benefits": []any{map[string]any{"benefit_id": foreignBenefit, "value": 1}}},
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			w := testSrv.PUT(planBenefitsPath(planID), body, adminHeader())
			assert.Equal(t, 422, w.Code, w.Body.String())
		})
	}
}

func TestPlanBenefits_Replace_Success_ReplaceAllIdempotent(t *testing.T) {
	_, planID, maxEmpID, payrollID := catalogFixture(t)
	body := map[string]any{"benefits": []any{
		map[string]any{"benefit_id": maxEmpID, "value": 200},
		map[string]any{"benefit_id": payrollID, "value": true},
	}}

	for i := 0; i < 2; i++ { // PUT kedua dengan body sama menghasilkan state sama
		w := testSrv.PUT(planBenefitsPath(planID), body, adminHeader())
		require.Equal(t, 200, w.Code, w.Body.String())
		got := decodeAssigned(t, w.Body.Bytes())
		require.Len(t, got, 2)
		assert.Equal(t, "max_employees", got[0].Code)
		assert.JSONEq(t, `200`, string(got[0].Value))
		assert.Equal(t, "payroll_module", got[1].Code)
		assert.JSONEq(t, `true`, string(got[1].Value))
	}

	// Replace-all: hanya payroll tersisa.
	w := testSrv.PUT(planBenefitsPath(planID), map[string]any{"benefits": []any{
		map[string]any{"benefit_id": payrollID, "value": false},
	}}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	got := decodeAssigned(t, w.Body.Bytes())
	require.Len(t, got, 1)
	assert.Equal(t, "payroll_module", got[0].Code)

	// Array kosong menghapus semua.
	w = testSrv.PUT(planBenefitsPath(planID), map[string]any{"benefits": []any{}}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Empty(t, decodeAssigned(t, w.Body.Bytes()))
}

func TestPlanBenefits_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(planBenefitsPath(uuid.NewString())).Code)
}

func TestPlanBenefits_List_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.GET(planBenefitsPath(uuid.NewString()), userHeader()).Code)
}

func TestPlanBenefits_List_InactivePlanHiddenFromNonAdmin(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("pl"))
	planID := mustCreatePlan(t, solID, "legacy", map[string]any{"is_active": false})
	assert.Equal(t, 404, testSrv.GET(planBenefitsPath(planID), userHeader()).Code)
	assert.Equal(t, 200, testSrv.GET(planBenefitsPath(planID), adminHeader()).Code)
}

func TestPlanBenefits_List_Success(t *testing.T) {
	_, planID, maxEmpID, _ := catalogFixture(t)
	require.Equal(t, 200, testSrv.PUT(planBenefitsPath(planID), map[string]any{"benefits": []any{
		map[string]any{"benefit_id": maxEmpID, "value": 50},
	}}, adminHeader()).Code)

	w := testSrv.GET(planBenefitsPath(planID), userHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	got := decodeAssigned(t, w.Body.Bytes())
	require.Len(t, got, 1)
	assert.Equal(t, maxEmpID, got[0].BenefitID)
	assert.Equal(t, "integer", got[0].ValueType)
	assert.JSONEq(t, `50`, string(got[0].Value))
}
