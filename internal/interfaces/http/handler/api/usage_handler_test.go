package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/testhelper"
)

func usagePath(tenantID, solutionID string) string {
	return "/api/v1/tenants/" + tenantID + "/solutions/" + solutionID + "/usage"
}

type usageData struct {
	BenefitCode string     `json:"benefit_code"`
	Mode        string     `json:"mode"`
	PeriodStart time.Time  `json:"period_start"`
	PeriodEnd   *time.Time `json:"period_end"`
	Used        string     `json:"used"`
	Replayed    bool       `json:"replayed"`
}

// usageFixture: solution dengan benefit counter (monthly & billing_period), gauge, dan
// non-meter; tenant subscribe; token service ber-scope usage:write.
type usageFixture struct {
	solutionID, tenantID, subscriptionID string
	auth                                 map[string]string
}

func newUsageFixture(t *testing.T) usageFixture {
	t.Helper()
	f := usageFixture{solutionID: mustCreateSolution(t, testhelper.UniqueCode("use"))}
	mustCreateBenefit(t, f.solutionID, "api_calls", "integer", "sum", map[string]any{"metering": "counter", "usage_reset": "monthly"})
	mustCreateBenefit(t, f.solutionID, "transactions", "integer", "sum", map[string]any{"metering": "counter", "usage_reset": "billing_period"})
	mustCreateBenefit(t, f.solutionID, "lifetime_exports", "integer", "sum", map[string]any{"metering": "counter"})
	mustCreateBenefit(t, f.solutionID, "active_employees", "integer", "max", map[string]any{"metering": "gauge"})
	mustCreateBenefit(t, f.solutionID, "export_pdf", "boolean", "any_true", nil)
	f.tenantID = mustCreateTenant(t, adminHeader())
	f.subscriptionID = mustSubscribe(t, f.tenantID, mustCreatePlan(t, f.solutionID, "basic", nil))
	f.auth = testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "usage:write"), ""))
	return f
}

func (f usageFixture) delta(code string, qty any, key string, extra map[string]any) map[string]any {
	body := map[string]any{"benefit_code": code, "quantity": qty, "idempotency_key": key}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func decodeUsage(t *testing.T, w interface{ Bytes() []byte }) usageData {
	t.Helper()
	var resp struct {
		Data usageData `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Bytes(), &resp))
	return resp.Data
}

// ── Akses ────────────────────────────────────────────────────────────────────

func TestUsage_Report_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(usagePath(uuid.NewString(), uuid.NewString()), map[string]any{}).Code)
}

func TestUsage_Report_Forbidden(t *testing.T) {
	f := newUsageFixture(t)
	body := f.delta("api_calls", 1, uuid.NewString(), nil)

	assert.Equal(t, 403, testSrv.POST(usagePath(f.tenantID, f.solutionID), body, adminHeader()).Code, "token user")
	noScope := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "entitlements:read"), ""))
	assert.Equal(t, 403, testSrv.POST(usagePath(f.tenantID, f.solutionID), body, noScope).Code, "tanpa usage:write")

	other := newUsageFixture(t)
	assert.Equal(t, 403, testSrv.POST(usagePath(f.tenantID, other.solutionID), body, f.auth).Code, "solution path bukan milik client")

	unsubscribed := mustCreateTenant(t, adminHeader())
	assert.Equal(t, 403, testSrv.POST(usagePath(unsubscribed, f.solutionID), body, f.auth).Code, "tenant tidak subscribe")
}

// ── Validasi ─────────────────────────────────────────────────────────────────

func TestUsage_Report_BenefitNotFound(t *testing.T) {
	f := newUsageFixture(t)
	w := testSrv.POST(usagePath(f.tenantID, f.solutionID), f.delta("tidak_ada", 1, uuid.NewString(), nil), f.auth)
	assert.Equal(t, 404, w.Code, w.Body.String())
}

func TestUsage_Report_Invalid(t *testing.T) {
	f := newUsageFixture(t)
	path := usagePath(f.tenantID, f.solutionID)
	tests := map[string]map[string]any{
		"benefit tidak di-meter": f.delta("export_pdf", 1, uuid.NewString(), nil),
		"gauge lewat POST":       f.delta("active_employees", 1, uuid.NewString(), nil),
		"delta nol":              f.delta("api_calls", 0, uuid.NewString(), nil),
		"delta negatif":          f.delta("api_calls", -3, uuid.NewString(), nil),
		"5 desimal":              f.delta("api_calls", "1.23456", uuid.NewString(), nil),
		"tanpa idempotency_key":  {"benefit_code": "api_calls", "quantity": 1},
		"tanpa quantity":         {"benefit_code": "api_calls", "idempotency_key": "k"},
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			w := testSrv.POST(path, body, f.auth)
			assert.Equal(t, 422, w.Code, w.Body.String())
		})
	}

	w := testSrv.PUT(path+"/api_calls", map[string]any{"quantity": 5, "idempotency_key": uuid.NewString()}, f.auth)
	assert.Equal(t, 422, w.Code, "counter lewat PUT: %s", w.Body.String())
}

// ── Counter (delta) ──────────────────────────────────────────────────────────

func TestUsage_Delta_AccumulatesAndIsIdempotent(t *testing.T) {
	f := newUsageFixture(t)
	path := usagePath(f.tenantID, f.solutionID)
	at := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	key1 := uuid.NewString()

	w := testSrv.POST(path, f.delta("api_calls", 5, key1, map[string]any{"occurred_at": at}), f.auth)
	require.Equal(t, 201, w.Code, w.Body.String())
	first := decodeUsage(t, w.Body)
	assert.Equal(t, "5.0000", first.Used)
	assert.Equal(t, "delta", first.Mode)
	assert.False(t, first.Replayed)
	assert.Equal(t, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), first.PeriodStart.UTC())
	require.NotNil(t, first.PeriodEnd)
	assert.Equal(t, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), first.PeriodEnd.UTC())

	w = testSrv.POST(path, f.delta("api_calls", "2.5", uuid.NewString(), map[string]any{"occurred_at": at}), f.auth)
	require.Equal(t, 201, w.Code, w.Body.String())
	assert.Equal(t, "7.5000", decodeUsage(t, w.Body).Used)

	// Replay key yang sama + payload sama → 200, tidak ditambah lagi.
	w = testSrv.POST(path, f.delta("api_calls", 5, key1, map[string]any{"occurred_at": at}), f.auth)
	require.Equal(t, 200, w.Code, w.Body.String())
	replay := decodeUsage(t, w.Body)
	assert.True(t, replay.Replayed)
	assert.Equal(t, "7.5000", replay.Used)

	// Key sama, payload beda → 409.
	w = testSrv.POST(path, f.delta("api_calls", 9, key1, nil), f.auth)
	assert.Equal(t, 409, w.Code, w.Body.String())

	var records int
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM usage_records WHERE tenant_id = $1`, f.tenantID).Scan(&records))
	assert.Equal(t, 2, records)
}

func TestUsage_Delta_MonthlyResetsByPeriod(t *testing.T) {
	f := newUsageFixture(t)
	path := usagePath(f.tenantID, f.solutionID)
	may := time.Date(2026, 5, 31, 23, 0, 0, 0, time.UTC)
	june := time.Date(2026, 6, 1, 1, 0, 0, 0, time.UTC)

	require.Equal(t, 201, testSrv.POST(path, f.delta("api_calls", 10, uuid.NewString(), map[string]any{"occurred_at": may}), f.auth).Code)
	w := testSrv.POST(path, f.delta("api_calls", 1, uuid.NewString(), map[string]any{"occurred_at": june}), f.auth)
	require.Equal(t, 201, w.Code)
	assert.Equal(t, "1.0000", decodeUsage(t, w.Body).Used, "periode Juni mulai dari 0")
}

func TestUsage_Delta_NeverReset_AndBillingPeriod(t *testing.T) {
	f := newUsageFixture(t)
	path := usagePath(f.tenantID, f.solutionID)

	w := testSrv.POST(path, f.delta("lifetime_exports", 3, uuid.NewString(), nil), f.auth)
	require.Equal(t, 201, w.Code, w.Body.String())
	never := decodeUsage(t, w.Body)
	assert.Equal(t, time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), never.PeriodStart.UTC())
	assert.Nil(t, never.PeriodEnd)

	w = testSrv.POST(path, f.delta("transactions", 1, uuid.NewString(), nil), f.auth)
	require.Equal(t, 201, w.Code, w.Body.String())
	billing := decodeUsage(t, w.Body)
	var start, end time.Time
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT current_period_start, current_period_end FROM subscriptions WHERE id = $1`, f.subscriptionID).Scan(&start, &end))
	assert.True(t, start.Equal(billing.PeriodStart), "periode = periode subscription")
	require.NotNil(t, billing.PeriodEnd)
	assert.True(t, end.Equal(*billing.PeriodEnd))
}

func TestUsage_Delta_ConcurrentReportsAreAtomic(t *testing.T) {
	f := newUsageFixture(t)
	path := usagePath(f.tenantID, f.solutionID)

	const n = 20
	var wg sync.WaitGroup
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i] = testSrv.POST(path, f.delta("lifetime_exports", 1, fmt.Sprintf("par-%s-%d", f.tenantID, i), nil), f.auth).Code
		}(i)
	}
	wg.Wait()
	for _, c := range codes {
		assert.Equal(t, 201, c)
	}

	var used string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT used::text FROM tenant_usage WHERE tenant_id = $1`, f.tenantID).Scan(&used))
	assert.Equal(t, "20.0000", used)
}

// ── Gauge (absolute) ─────────────────────────────────────────────────────────

func TestUsage_Absolute_LatestWins(t *testing.T) {
	f := newUsageFixture(t)
	path := usagePath(f.tenantID, f.solutionID) + "/active_employees"
	t1 := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)

	w := testSrv.PUT(path, map[string]any{"quantity": 120, "idempotency_key": uuid.NewString(), "occurred_at": t1}, f.auth)
	require.Equal(t, 200, w.Code, w.Body.String())
	got := decodeUsage(t, w.Body)
	assert.Equal(t, "120.0000", got.Used)
	assert.Equal(t, "absolute", got.Mode)
	assert.Nil(t, got.PeriodEnd, "gauge tidak di-reset")

	// Laporan lebih lama (datang terlambat) tidak menimpa.
	w = testSrv.PUT(path, map[string]any{"quantity": 100, "idempotency_key": uuid.NewString(), "occurred_at": t1.Add(-time.Hour)}, f.auth)
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, "120.0000", decodeUsage(t, w.Body).Used)

	// Laporan lebih baru menimpa, termasuk turun ke 0.
	w = testSrv.PUT(path, map[string]any{"quantity": "0", "idempotency_key": uuid.NewString(), "occurred_at": t1.Add(time.Hour)}, f.auth)
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, "0.0000", decodeUsage(t, w.Body).Used)
}

func TestUsage_Absolute_Replay(t *testing.T) {
	f := newUsageFixture(t)
	path := usagePath(f.tenantID, f.solutionID) + "/active_employees"
	body := map[string]any{"quantity": 7, "idempotency_key": uuid.NewString()}

	require.Equal(t, 200, testSrv.PUT(path, body, f.auth).Code)
	w := testSrv.PUT(path, body, f.auth)
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.True(t, decodeUsage(t, w.Body).Replayed)
}
