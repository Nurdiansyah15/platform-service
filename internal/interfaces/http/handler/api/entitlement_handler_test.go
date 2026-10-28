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

func entitlementsPath(companyID, solutionID string) string {
	return "/api/v1/companies/" + companyID + "/solutions/" + solutionID + "/entitlements"
}

type entitlementData struct {
	BenefitCode  string          `json:"benefit_code"`
	Granted      bool            `json:"granted"`
	Value        json.RawMessage `json:"value"`
	Unlimited    bool            `json:"unlimited"`
	Limit        *string         `json:"limit"`
	Used         *string         `json:"used"`
	Remaining    *string         `json:"remaining"`
	LimitReached bool            `json:"limit_reached"`
}

type entitlementsData struct {
	Subscription *struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	} `json:"subscription"`
	Entitlements []entitlementData `json:"entitlements"`
}

func getEntitlements(t *testing.T, companyID, solutionID string, h map[string]string) (entitlementsData, map[string]entitlementData) {
	t.Helper()
	w := testSrv.GET(entitlementsPath(companyID, solutionID), h)
	require.Equal(t, 200, w.Code, w.Body.String())
	var got entitlementsData
	testhelper.DecodeBody(t, w, &got)
	byCode := map[string]entitlementData{}
	for _, e := range got.Entitlements {
		byCode[e.BenefitCode] = e
	}
	return got, byCode
}

// ── Akses ────────────────────────────────────────────────────────────────────

func TestEntitlement_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(entitlementsPath(uuid.NewString(), uuid.NewString())).Code)
}

func TestEntitlement_List_Forbidden(t *testing.T) {
	f := newSubFixture(t)
	mustSubscribe(t, f.companyID, f.basicPlan)

	strangerH, _, _ := mustCompanyOwner(t)
	assert.Equal(t, 403, testSrv.GET(entitlementsPath(f.companyID, f.solutionID), strangerH).Code, "user company lain")

	other := newSubFixture(t)
	foreign := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, other.solutionID, "entitlements:read"), ""))
	assert.Equal(t, 403, testSrv.GET(entitlementsPath(f.companyID, f.solutionID), foreign).Code, "service solution lain")

	noScope := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "usage:write"), ""))
	assert.Equal(t, 403, testSrv.GET(entitlementsPath(f.companyID, f.solutionID), noScope).Code, "tanpa entitlements:read")
}

func TestEntitlement_List_NotFound(t *testing.T) {
	f := newSubFixture(t)
	assert.Equal(t, 404, testSrv.GET(entitlementsPath(uuid.NewString(), f.solutionID), adminHeader()).Code, "company")
	assert.Equal(t, 404, testSrv.GET(entitlementsPath(f.companyID, uuid.NewString()), adminHeader()).Code, "solution")
	assert.Equal(t, 404, testSrv.GET(entitlementsPath(f.companyID, f.solutionID)+"/tidak_ada", adminHeader()).Code, "benefit")
	assert.Equal(t, 422, testSrv.GET(entitlementsPath("bukan-uuid", f.solutionID), adminHeader()).Code)
}

// ── Perhitungan ──────────────────────────────────────────────────────────────

func TestEntitlement_List_NoSubscription(t *testing.T) {
	f := newSubFixture(t)
	got, byCode := getEntitlements(t, f.companyID, f.solutionID, adminHeader())
	assert.Nil(t, got.Subscription)
	require.Len(t, got.Entitlements, 2)
	assert.False(t, byCode["max_employees"].Granted)
	assert.JSONEq(t, `null`, string(byCode["max_employees"].Value))
}

func TestEntitlement_List_PlanPlusAddonQuantity(t *testing.T) {
	f := newSubFixture(t)
	subID := mustSubscribe(t, f.companyID, f.basicPlan) // basic: max_employees=10, tanpa payroll
	addonID := addonFixture(t, f)                       // extra-employees: max_employees=50
	require.Equal(t, 201, testSrv.POST(subscriptionPath(subID, "addons"), map[string]any{"addon_id": addonID, "quantity": 2}, adminHeader()).Code)

	got, byCode := getEntitlements(t, f.companyID, f.solutionID, adminHeader())
	require.NotNil(t, got.Subscription)
	assert.Equal(t, subID, got.Subscription.ID)

	assert.True(t, byCode["max_employees"].Granted)
	assert.JSONEq(t, `110`, string(byCode["max_employees"].Value), "10 + 50 × 2")
	assert.False(t, byCode["payroll_module"].Granted)

	// Add-on dibatalkan → tidak dihitung lagi.
	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE subscription_addons SET status = 'canceled' WHERE subscription_id = $1`, subID)
	require.NoError(t, err)
	_, byCode = getEntitlements(t, f.companyID, f.solutionID, adminHeader())
	assert.JSONEq(t, `10`, string(byCode["max_employees"].Value))
}

func TestEntitlement_List_Unlimited(t *testing.T) {
	f := newSubFixture(t)
	enterprise := mustCreatePlan(t, f.solutionID, "enterprise", nil)
	mustReplacePlanBenefits(t, enterprise, map[string]any{f.maxEmp: -1, f.payroll: true})
	mustSubscribe(t, f.companyID, enterprise)

	_, byCode := getEntitlements(t, f.companyID, f.solutionID, adminHeader())
	assert.True(t, byCode["max_employees"].Unlimited)
	assert.JSONEq(t, `-1`, string(byCode["max_employees"].Value))
	assert.Nil(t, byCode["max_employees"].Limit)
	assert.JSONEq(t, `true`, string(byCode["payroll_module"].Value))
}

func TestEntitlement_List_CanceledSubscriptionNotGranted(t *testing.T) {
	f := newSubFixture(t)
	subID := mustSubscribe(t, f.companyID, f.proPlan)
	require.Equal(t, 200, testSrv.POST(subscriptionPath(subID, "cancel"), nil, adminHeader()).Code)

	got, byCode := getEntitlements(t, f.companyID, f.solutionID, adminHeader())
	assert.Nil(t, got.Subscription)
	assert.False(t, byCode["payroll_module"].Granted)
}

func TestEntitlement_AccessibleByMemberAndService(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	var solutionID string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT solution_id FROM subscriptions WHERE company_id = $1`, companyID).Scan(&solutionID))

	assert.Equal(t, 200, testSrv.GET(entitlementsPath(companyID, solutionID), h).Code)
	svc := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, solutionID, "entitlements:read"), ""))
	assert.Equal(t, 200, testSrv.GET(entitlementsPath(companyID, solutionID), svc).Code)
}

// ── Metered: limit / used / remaining & usage.limit_reached ──────────────────

func TestEntitlement_Metered_UsedRemainingAndLimitReachedEvent(t *testing.T) {
	f := newSubFixture(t)
	mustCreateBenefit(t, f.solutionID, "api_calls", "integer", "sum", map[string]any{"metering": "counter", "usage_reset": "monthly"})
	var apiCallsID string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT id FROM benefits WHERE solution_id = $1 AND code = 'api_calls'`, f.solutionID).Scan(&apiCallsID))
	mustReplacePlanBenefits(t, f.basicPlan, map[string]any{f.maxEmp: 10, apiCallsID: 100})
	mustSubscribe(t, f.companyID, f.basicPlan)

	svc := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "usage:write", "entitlements:read"), ""))
	report := func(qty int) usageData {
		w := testSrv.POST(usagePath(f.companyID, f.solutionID), map[string]any{
			"benefit_code": "api_calls", "quantity": qty, "idempotency_key": uuid.NewString(),
		}, svc)
		require.Equal(t, 201, w.Code, w.Body.String())
		var resp struct {
			Data struct {
				usageData
				Limit        *string `json:"limit"`
				Remaining    *string `json:"remaining"`
				LimitReached bool    `json:"limit_reached"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.NotNil(t, resp.Data.Limit)
		assert.Equal(t, "100", *resp.Data.Limit)
		return resp.Data.usageData
	}

	report(30)
	w := testSrv.GET(entitlementsPath(f.companyID, f.solutionID)+"/api_calls", svc)
	require.Equal(t, 200, w.Code, w.Body.String())
	var ent entitlementData
	testhelper.DecodeBody(t, w, &ent)
	require.NotNil(t, ent.Used)
	assert.Equal(t, "30", *ent.Used)
	assert.Equal(t, "70", *ent.Remaining)
	assert.False(t, ent.LimitReached)
	assert.Equal(t, 0, countEvents(t, "usage.limit_reached", f.companyID))

	report(70) // tepat mencapai limit → event
	report(5)  // sudah di atas limit → tidak ada event baru
	assert.Equal(t, 1, countEvents(t, "usage.limit_reached", f.companyID))

	_, byCode := getEntitlements(t, f.companyID, f.solutionID, svc)
	assert.Equal(t, "105", *byCode["api_calls"].Used)
	assert.Equal(t, "0", *byCode["api_calls"].Remaining)
	assert.True(t, byCode["api_calls"].LimitReached)
	// Benefit tidak di-meter tidak memuat used / remaining.
	assert.Nil(t, byCode["max_employees"].Used)
}
