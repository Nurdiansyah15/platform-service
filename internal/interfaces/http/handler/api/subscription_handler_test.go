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

func subscriptionsPath(companyID string) string {
	return "/api/v1/companies/" + companyID + "/subscriptions"
}
func subscriptionPath(id, action string) string { return "/api/v1/subscriptions/" + id + "/" + action }

type subscriptionData struct {
	ID               string `json:"id"`
	CompanyID        string `json:"company_id"`
	SolutionID       string `json:"solution_id"`
	PlanID           string `json:"plan_id"`
	PlanCode         string `json:"plan_code"`
	Status           string `json:"status"`
	BillingCycle     string `json:"billing_cycle"`
	PriceAtPurchase  string `json:"price_at_purchase"`
	CurrentPeriodEnd string `json:"current_period_end"`
}

// subFixture: solution dengan dua benefit, plan basic (max_employees=10) dan pro
// (max_employees=100, payroll=true), serta company kosong milik admin.
type subFixture struct {
	solutionID, basicPlan, proPlan, maxEmp, payroll, companyID string
}

func newSubFixture(t *testing.T) subFixture {
	t.Helper()
	f := subFixture{solutionID: mustCreateSolution(t, testhelper.UniqueCode("sub"))}
	f.maxEmp = mustCreateBenefit(t, f.solutionID, "max_employees", "integer", "sum", nil)
	f.payroll = mustCreateBenefit(t, f.solutionID, "payroll_module", "boolean", "any_true", nil)
	f.basicPlan = mustCreatePlan(t, f.solutionID, "basic", map[string]any{"price": "100000"})
	f.proPlan = mustCreatePlan(t, f.solutionID, "pro", map[string]any{"price": "300000", "billing_cycle": "yearly"})
	mustReplacePlanBenefits(t, f.basicPlan, map[string]any{f.maxEmp: 10})
	mustReplacePlanBenefits(t, f.proPlan, map[string]any{f.maxEmp: 100, f.payroll: true})
	f.companyID = mustCreateCompany(t, adminHeader())
	return f
}

func mustReplacePlanBenefits(t *testing.T, planID string, values map[string]any) {
	t.Helper()
	items := []any{}
	for id, v := range values {
		items = append(items, map[string]any{"benefit_id": id, "value": v})
	}
	w := testSrv.PUT("/api/v1/plans/"+planID+"/benefits", map[string]any{"benefits": items}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
}

// snapshotValues membaca snapshot subscription: map benefit_id → value (JSON string).
// addon=true → hanya snapshot add-on, false → hanya snapshot plan.
func snapshotValues(t *testing.T, subscriptionID string, addon bool) map[string]string {
	t.Helper()
	cond := "subscription_addon_id IS NULL"
	if addon {
		cond = "subscription_addon_id IS NOT NULL"
	}
	rows, err := testSrv.DB.QueryContext(context.Background(),
		`SELECT benefit_id, value::text FROM subscription_benefit_snapshots WHERE subscription_id = $1 AND `+cond, subscriptionID)
	require.NoError(t, err)
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, v string
		require.NoError(t, rows.Scan(&id, &v))
		out[id] = v
	}
	require.NoError(t, rows.Err())
	return out
}

func eventTypesFor(t *testing.T, companyID string) []string {
	t.Helper()
	rows, err := testSrv.DB.QueryContext(context.Background(),
		`SELECT event_type FROM events WHERE company_id = $1 ORDER BY occurred_at, event_type`, companyID)
	require.NoError(t, err)
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var e string
		require.NoError(t, rows.Scan(&e))
		out = append(out, e)
	}
	return out
}

// ── Create ───────────────────────────────────────────────────────────────────

func TestSubscription_Create_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(subscriptionsPath(uuid.NewString()), map[string]any{"plan_id": uuid.NewString()}).Code)
}

func TestSubscription_Create_Forbidden_NonAdmin(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	assert.Equal(t, 403, testSrv.POST(subscriptionsPath(companyID), map[string]any{"plan_id": uuid.NewString()}, h).Code)
}

func TestSubscription_Create_Forbidden_LicenseManaged(t *testing.T) {
	f := newSubFixture(t)
	w := testOnPremSrv.POST(subscriptionsPath(f.companyID), map[string]any{"plan_id": f.basicPlan}, adminHeader())
	require.Equal(t, 403, w.Code, w.Body.String())
	assert.Equal(t, "LICENSE_MANAGED", testhelper.DecodeError(t, w).ErrorCode)
}

func TestSubscription_Create_InvalidPayload(t *testing.T) {
	f := newSubFixture(t)
	inactivePlan := mustCreatePlan(t, f.solutionID, "legacy", map[string]any{"is_active": false})
	tests := map[string]map[string]any{
		"tanpa plan_id":      {},
		"plan_id bukan uuid": {"plan_id": "x"},
		"status canceled":    {"plan_id": f.basicPlan, "status": "canceled"},
		"plan tak dikenal":   {"plan_id": uuid.NewString()},
		"plan tidak aktif":   {"plan_id": inactivePlan},
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, 422, testSrv.POST(subscriptionsPath(f.companyID), body, adminHeader()).Code)
		})
	}
}

func TestSubscription_Create_CompanyNotFound(t *testing.T) {
	f := newSubFixture(t)
	assert.Equal(t, 404, testSrv.POST(subscriptionsPath(uuid.NewString()), map[string]any{"plan_id": f.basicPlan}, adminHeader()).Code)
}

func TestSubscription_Create_CompanySuspended(t *testing.T) {
	f := newSubFixture(t)
	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE companies SET status = 'suspended' WHERE id = $1`, f.companyID)
	require.NoError(t, err)
	assert.Equal(t, 422, testSrv.POST(subscriptionsPath(f.companyID), map[string]any{"plan_id": f.basicPlan}, adminHeader()).Code)
}

func TestSubscription_Create_AlreadyLive(t *testing.T) {
	f := newSubFixture(t)
	mustSubscribe(t, f.companyID, f.basicPlan)
	w := testSrv.POST(subscriptionsPath(f.companyID), map[string]any{"plan_id": f.proPlan}, adminHeader())
	assert.Equal(t, 409, w.Code, w.Body.String())
}

func TestSubscription_Create_Success_SnapshotAndEvents(t *testing.T) {
	f := newSubFixture(t)
	w := testSrv.POST(subscriptionsPath(f.companyID), map[string]any{
		"plan_id": f.proPlan, "status": "trialing", "current_period_start": "2026-01-31T00:00:00Z",
	}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())

	var got subscriptionData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, f.companyID, got.CompanyID)
	assert.Equal(t, f.solutionID, got.SolutionID)
	assert.Equal(t, "pro", got.PlanCode)
	assert.Equal(t, "trialing", got.Status)
	assert.Equal(t, "yearly", got.BillingCycle)
	assert.Equal(t, "300000.00", got.PriceAtPurchase)
	assert.Contains(t, got.CurrentPeriodEnd, "2027-01-31")

	assert.Equal(t, map[string]string{f.maxEmp: "100", f.payroll: "true"}, snapshotValues(t, got.ID, false))
	assert.ElementsMatch(t, []string{"entitlement.changed", "subscription.created"}, eventTypesFor(t, f.companyID))
}

func TestSubscription_Snapshot_UnaffectedByPlanChanges(t *testing.T) {
	f := newSubFixture(t)
	subID := mustSubscribe(t, f.companyID, f.basicPlan)

	// Plan master diubah setelah subscribe → snapshot subscription tetap.
	mustReplacePlanBenefits(t, f.basicPlan, map[string]any{f.maxEmp: 999, f.payroll: true})
	assert.Equal(t, map[string]string{f.maxEmp: "10"}, snapshotValues(t, subID, false))
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestSubscription_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(subscriptionsPath(uuid.NewString())).Code)
}

func TestSubscription_List_Forbidden_OtherCompany(t *testing.T) {
	_, _, companyID := mustCompanyOwner(t)
	strangerH, _, _ := mustCompanyOwner(t)
	assert.Equal(t, 403, testSrv.GET(subscriptionsPath(companyID), strangerH).Code)
}

func TestSubscription_List_Success_Member(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	w := testSrv.GET(subscriptionsPath(companyID), h)
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp struct {
		Data []subscriptionData `json:"data"`
		Meta struct {
			Total int64 `json:"total"`
		} `json:"meta"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "basic", resp.Data[0].PlanCode)
	assert.Equal(t, int64(1), resp.Meta.Total)
}

func TestSubscription_List_ServiceToken_OnlyOwnSolution(t *testing.T) {
	f := newSubFixture(t)
	mustSubscribe(t, f.companyID, f.basicPlan)
	other := newSubFixture(t)
	// Company f juga subscribe solution lain.
	mustSubscribe(t, f.companyID, other.basicPlan)

	c := mustCreateClient(t, f.solutionID, "entitlements:read")
	w := testSrv.GET(subscriptionsPath(f.companyID), testhelper.BearerHeader(mustServiceToken(t, c, "")))
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp struct {
		Data []subscriptionData `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, f.solutionID, resp.Data[0].SolutionID)

	// Admin melihat keduanya.
	w = testSrv.GET(subscriptionsPath(f.companyID), adminHeader())
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp.Data, 2)
}

func TestSubscription_List_ServiceToken_Forbidden(t *testing.T) {
	f := newSubFixture(t) // company belum subscribe
	c := mustCreateClient(t, f.solutionID, "entitlements:read", "users:read")
	assert.Equal(t, 403, testSrv.GET(subscriptionsPath(f.companyID), testhelper.BearerHeader(mustServiceToken(t, c, ""))).Code,
		"company tidak subscribe solution client")

	mustSubscribe(t, f.companyID, f.basicPlan)
	assert.Equal(t, 403, testSrv.GET(subscriptionsPath(f.companyID), testhelper.BearerHeader(mustServiceToken(t, c, "users:read"))).Code,
		"scope entitlements:read tidak ada")
}

// ── Change plan ──────────────────────────────────────────────────────────────

func TestSubscription_ChangePlan_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(subscriptionPath(uuid.NewString(), "change-plan"), map[string]any{"plan_id": uuid.NewString()}).Code)
}

func TestSubscription_ChangePlan_Forbidden_NonAdmin(t *testing.T) {
	h, _, _ := mustCompanyOwner(t)
	assert.Equal(t, 403, testSrv.POST(subscriptionPath(uuid.NewString(), "change-plan"), map[string]any{"plan_id": uuid.NewString()}, h).Code)
}

func TestSubscription_ChangePlan_NotFound(t *testing.T) {
	f := newSubFixture(t)
	assert.Equal(t, 404, testSrv.POST(subscriptionPath(uuid.NewString(), "change-plan"), map[string]any{"plan_id": f.proPlan}, adminHeader()).Code)
}

func TestSubscription_ChangePlan_Invalid(t *testing.T) {
	f := newSubFixture(t)
	subID := mustSubscribe(t, f.companyID, f.basicPlan)
	other := newSubFixture(t)

	assert.Equal(t, 422, testSrv.POST(subscriptionPath(subID, "change-plan"), map[string]any{"plan_id": f.basicPlan}, adminHeader()).Code, "plan sama")
	assert.Equal(t, 422, testSrv.POST(subscriptionPath(subID, "change-plan"), map[string]any{"plan_id": other.proPlan}, adminHeader()).Code, "plan solution lain")
	assert.Equal(t, 422, testSrv.POST(subscriptionPath(subID, "change-plan"), map[string]any{}, adminHeader()).Code, "tanpa plan_id")
}

func TestSubscription_ChangePlan_Success_ReplacesPlanSnapshot(t *testing.T) {
	f := newSubFixture(t)
	subID := mustSubscribe(t, f.companyID, f.basicPlan)

	w := testSrv.POST(subscriptionPath(subID, "change-plan"), map[string]any{"plan_id": f.proPlan}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got subscriptionData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "pro", got.PlanCode)
	assert.Equal(t, "300000.00", got.PriceAtPurchase)
	assert.Equal(t, "yearly", got.BillingCycle)

	assert.Equal(t, map[string]string{f.maxEmp: "100", f.payroll: "true"}, snapshotValues(t, subID, false))
	assert.Contains(t, eventTypesFor(t, f.companyID), "subscription.plan_changed")
}

// ── Cancel ───────────────────────────────────────────────────────────────────

func TestSubscription_Cancel_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(subscriptionPath(uuid.NewString(), "cancel"), nil).Code)
}

func TestSubscription_Cancel_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.POST(subscriptionPath(uuid.NewString(), "cancel"), nil, adminHeader()).Code)
}

func TestSubscription_Cancel_Success_AndConflicts(t *testing.T) {
	f := newSubFixture(t)
	subID := mustSubscribe(t, f.companyID, f.basicPlan)

	w := testSrv.POST(subscriptionPath(subID, "cancel"), nil, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got subscriptionData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "canceled", got.Status)
	assert.Contains(t, eventTypesFor(t, f.companyID), "subscription.status_changed")
	assert.NotEmpty(t, snapshotValues(t, subID, false), "snapshot tetap disimpan sebagai histori")

	assert.Equal(t, 409, testSrv.POST(subscriptionPath(subID, "cancel"), nil, adminHeader()).Code, "cancel dua kali")
	assert.Equal(t, 409, testSrv.POST(subscriptionPath(subID, "change-plan"), map[string]any{"plan_id": f.proPlan}, adminHeader()).Code,
		"ganti plan subscription yang sudah dibatalkan")

	// Subscription baru boleh dibuat setelah yang lama dibatalkan.
	mustSubscribe(t, f.companyID, f.proPlan)
}

// ── Add-ons ──────────────────────────────────────────────────────────────────

func addonFixture(t *testing.T, f subFixture) string {
	t.Helper()
	addonID := mustCreateAddon(t, f.solutionID, "extra-employees", map[string]any{"price": "25000"})
	w := testSrv.PUT("/api/v1/addons/"+addonID+"/benefits", map[string]any{"benefits": []any{
		map[string]any{"benefit_id": f.maxEmp, "value": 50},
	}}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	return addonID
}

func TestSubscriptionAddon_Add_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(subscriptionPath(uuid.NewString(), "addons"), map[string]any{"addon_id": uuid.NewString()}).Code)
}

func TestSubscriptionAddon_Add_Forbidden_NonAdmin(t *testing.T) {
	h, _, _ := mustCompanyOwner(t)
	assert.Equal(t, 403, testSrv.POST(subscriptionPath(uuid.NewString(), "addons"), map[string]any{"addon_id": uuid.NewString()}, h).Code)
}

func TestSubscriptionAddon_Add_Forbidden_LicenseManaged(t *testing.T) {
	f := newSubFixture(t)
	subID := mustSubscribe(t, f.companyID, f.basicPlan)
	assert.Equal(t, 403, testOnPremSrv.POST(subscriptionPath(subID, "addons"), map[string]any{"addon_id": addonFixture(t, f)}, adminHeader()).Code)
}

func TestSubscriptionAddon_Add_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.POST(subscriptionPath(uuid.NewString(), "addons"), map[string]any{"addon_id": uuid.NewString()}, adminHeader()).Code)
}

func TestSubscriptionAddon_Add_Invalid(t *testing.T) {
	f := newSubFixture(t)
	subID := mustSubscribe(t, f.companyID, f.basicPlan)
	addonID := addonFixture(t, f)
	otherAddon := mustCreateAddon(t, newSubFixture(t).solutionID, "x", nil)

	assert.Equal(t, 422, testSrv.POST(subscriptionPath(subID, "addons"), map[string]any{"addon_id": addonID, "quantity": 0}, adminHeader()).Code, "quantity 0")
	assert.Equal(t, 422, testSrv.POST(subscriptionPath(subID, "addons"), map[string]any{"addon_id": otherAddon}, adminHeader()).Code, "add-on solution lain")
	assert.Equal(t, 422, testSrv.POST(subscriptionPath(subID, "addons"), map[string]any{"addon_id": uuid.NewString()}, adminHeader()).Code, "add-on tak dikenal")
}

func TestSubscriptionAddon_Add_Success_DuplicateAndReactivate(t *testing.T) {
	f := newSubFixture(t)
	subID := mustSubscribe(t, f.companyID, f.basicPlan)
	addonID := addonFixture(t, f)

	w := testSrv.POST(subscriptionPath(subID, "addons"), map[string]any{"addon_id": addonID, "quantity": 2}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())
	var got struct {
		ID                  string `json:"id"`
		AddonCode           string `json:"addon_code"`
		Quantity            int    `json:"quantity"`
		UnitPriceAtPurchase string `json:"unit_price_at_purchase"`
		Status              string `json:"status"`
	}
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "extra-employees", got.AddonCode)
	assert.Equal(t, 2, got.Quantity)
	assert.Equal(t, "25000.00", got.UnitPriceAtPurchase)
	assert.Equal(t, "active", got.Status)

	assert.Equal(t, map[string]string{f.maxEmp: "50"}, snapshotValues(t, subID, true))
	assert.Equal(t, map[string]string{f.maxEmp: "10"}, snapshotValues(t, subID, false), "snapshot plan tidak tersentuh")
	assert.Contains(t, eventTypesFor(t, f.companyID), "subscription_addon.added")

	assert.Equal(t, 409, testSrv.POST(subscriptionPath(subID, "addons"), map[string]any{"addon_id": addonID}, adminHeader()).Code, "sudah aktif")

	// Dibatalkan (DELETE add-on menyusul di L4) lalu dipasang ulang → baris yang sama aktif lagi.
	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE subscription_addons SET status = 'canceled' WHERE id = $1`, got.ID)
	require.NoError(t, err)
	w = testSrv.POST(subscriptionPath(subID, "addons"), map[string]any{"addon_id": addonID, "quantity": 3}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())
	var again struct {
		ID       string `json:"id"`
		Quantity int    `json:"quantity"`
	}
	testhelper.DecodeBody(t, w, &again)
	assert.Equal(t, got.ID, again.ID)
	assert.Equal(t, 3, again.Quantity)
}

func TestSubscriptionAddon_Add_CanceledSubscription(t *testing.T) {
	f := newSubFixture(t)
	subID := mustSubscribe(t, f.companyID, f.basicPlan)
	require.Equal(t, 200, testSrv.POST(subscriptionPath(subID, "cancel"), nil, adminHeader()).Code)
	assert.Equal(t, 409, testSrv.POST(subscriptionPath(subID, "addons"), map[string]any{"addon_id": addonFixture(t, f)}, adminHeader()).Code)
}

func TestSubscriptionAddon_List(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	var subID string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(), `SELECT id FROM subscriptions WHERE company_id = $1`, companyID).Scan(&subID))

	assert.Equal(t, 401, testSrv.GET(subscriptionPath(subID, "addons")).Code)
	strangerH, _, _ := mustCompanyOwner(t)
	assert.Equal(t, 404, testSrv.GET(subscriptionPath(subID, "addons"), strangerH).Code, "subscription company lain disembunyikan")
	assert.Equal(t, 404, testSrv.GET(subscriptionPath(uuid.NewString(), "addons"), adminHeader()).Code)

	w := testSrv.GET(subscriptionPath(subID, "addons"), h)
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp struct {
		Data []any `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Empty(t, resp.Data)
}
