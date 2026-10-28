package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/webhook/service"
	"platform-service/internal/testhelper"
)

// unreachableHookURL — port 1 menolak koneksi seketika, jadi dispatcher di test lain
// tidak menunggu timeout untuk endpoint yang dibuat test handler.
const unreachableHookURL = "http://127.0.0.1:1/hook"

func webhooksPath(solutionID string) string { return "/api/v1/solutions/" + solutionID + "/webhooks" }

type webhookData struct {
	ID         string   `json:"id"`
	SolutionID string   `json:"solution_id"`
	URL        string   `json:"url"`
	EventTypes []string `json:"event_types"`
	Status     string   `json:"status"`
	Secret     string   `json:"secret"`
}

func mustCreateWebhook(t *testing.T, solutionID, hookURL string, eventTypes ...string) webhookData {
	t.Helper()
	w := testSrv.POST(webhooksPath(solutionID), map[string]any{"url": hookURL, "event_types": eventTypes}, adminHeader())
	require.Equal(t, 201, w.Code, "mustCreateWebhook body=%s", w.Body.String())
	var got webhookData
	testhelper.DecodeBody(t, w, &got)
	return got
}

// ── Create webhook ───────────────────────────────────────────────────────────

func TestWebhook_Create_Unauthenticated(t *testing.T) {
	w := testSrv.POST(webhooksPath(uuid.NewString()), map[string]any{"url": unreachableHookURL, "event_types": []string{"branch.created"}})
	assert.Equal(t, 401, w.Code)
}

func TestWebhook_Create_Forbidden_NonAdmin(t *testing.T) {
	solutionID := mustCreateSolution(t, testhelper.UniqueCode("wh"))
	w := testSrv.POST(webhooksPath(solutionID), map[string]any{"url": unreachableHookURL, "event_types": []string{"branch.created"}}, userHeader())
	assert.Equal(t, 403, w.Code)
}

func TestWebhook_Create_InvalidPayload(t *testing.T) {
	solutionID := mustCreateSolution(t, testhelper.UniqueCode("wh"))
	cases := map[string]map[string]any{
		"missing url":        {"event_types": []string{"branch.created"}},
		"missing events":     {"url": unreachableHookURL},
		"empty events":       {"url": unreachableHookURL, "event_types": []string{}},
		"unknown event type": {"url": unreachableHookURL, "event_types": []string{"company.exploded"}},
		"relative url":       {"url": "/hook", "event_types": []string{"branch.created"}},
		"ftp url":            {"url": "ftp://example.com/hook", "event_types": []string{"branch.created"}},
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			w := testSrv.POST(webhooksPath(solutionID), body, adminHeader())
			assert.True(t, w.Code == 400 || w.Code == 422, "code=%d body=%s", w.Code, w.Body.String())
		})
	}
}

func TestWebhook_Create_SolutionNotFound(t *testing.T) {
	w := testSrv.POST(webhooksPath(uuid.NewString()), map[string]any{"url": unreachableHookURL, "event_types": []string{"branch.created"}}, adminHeader())
	assert.Equal(t, 404, w.Code)
}

func TestWebhook_Create_Success_SecretShownOnceAndEncrypted(t *testing.T) {
	solutionID := mustCreateSolution(t, testhelper.UniqueCode("wh"))
	got := mustCreateWebhook(t, solutionID, unreachableHookURL, "subscription.created", "branch.created", "branch.created")

	assert.NotEmpty(t, got.ID)
	assert.Equal(t, solutionID, got.SolutionID)
	assert.Equal(t, unreachableHookURL, got.URL)
	assert.Equal(t, []string{"branch.created", "subscription.created"}, got.EventTypes, "unik & terurut")
	assert.Equal(t, "active", got.Status)
	assert.True(t, strings.HasPrefix(got.Secret, "whsec_"), got.Secret)

	var stored string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT secret_encrypted FROM webhook_endpoints WHERE id = $1`, got.ID).Scan(&stored))
	assert.NotEqual(t, got.Secret, stored, "secret disimpan terenkripsi")
	assert.NotContains(t, stored, strings.TrimPrefix(got.Secret, "whsec_"))
}

// ── Events (polling) ─────────────────────────────────────────────────────────

// eventsFixture: solution dengan company yang berlangganan + company (event company-level),
// client events:read, dan solution lain yang tidak dilanggani company.
type eventsFixture struct {
	sub          subFixture
	token        string
	otherToken   string
	otherCompany string
	subscription string
}

func newEventsFixture(t *testing.T) eventsFixture {
	t.Helper()
	f := eventsFixture{sub: newSubFixture(t)}
	f.subscription = mustSubscribe(t, f.sub.companyID, f.sub.basicPlan)
	mustCreateBranch(t, adminHeader(), f.sub.companyID, testhelper.UniqueCode("co"))
	f.token = mustServiceToken(t, mustCreateClient(t, f.sub.solutionID, "events:read"), "")

	other := mustCreateSolution(t, testhelper.UniqueCode("oth"))
	f.otherToken = mustServiceToken(t, mustCreateClient(t, other, "events:read"), "")
	// Company lain yang tidak berlangganan solution fixture.
	f.otherCompany = mustCreateCompany(t, adminHeader())
	mustCreateBranch(t, adminHeader(), f.otherCompany, testhelper.UniqueCode("co"))
	return f
}

type eventData struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	CompanyID  *string         `json:"company_id"`
	SolutionID *string         `json:"solution_id"`
	OccurredAt time.Time       `json:"occurred_at"`
	Data       json.RawMessage `json:"data"`
}

type eventsPage struct {
	Data []eventData `json:"data"`
	Meta struct {
		Limit      int     `json:"limit"`
		NextCursor *string `json:"next_cursor"`
	} `json:"meta"`
}

func listEvents(t *testing.T, token string, q url.Values) eventsPage {
	t.Helper()
	w := testSrv.GET("/api/v1/events?"+q.Encode(), testhelper.BearerHeader(token))
	require.Equal(t, 200, w.Code, w.Body.String())
	var page eventsPage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
	return page
}

func eventTypesOf(events []eventData) []string {
	out := make([]string, 0, len(events))
	for _, e := range events {
		out = append(out, e.Type)
	}
	return out
}

func TestEvent_List_Unauthenticated(t *testing.T) {
	w := testSrv.GET("/api/v1/events")
	assert.Equal(t, 401, w.Code)
}

func TestEvent_List_Forbidden(t *testing.T) {
	assert.Equal(t, 403, testSrv.GET("/api/v1/events", adminHeader()).Code, "token user")

	solutionID := mustCreateSolution(t, testhelper.UniqueCode("ev"))
	token := mustServiceToken(t, mustCreateClient(t, solutionID, "usage:write"), "")
	assert.Equal(t, 403, testSrv.GET("/api/v1/events", testhelper.BearerHeader(token)).Code, "scope kurang")
}

func TestEvent_List_InvalidQuery(t *testing.T) {
	solutionID := mustCreateSolution(t, testhelper.UniqueCode("ev"))
	h := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, solutionID, "events:read"), ""))
	for _, q := range []string{"company_id=bukan-uuid", "limit=0", "limit=201", "since=kemarin", "cursor=!!!", "cursor=" + url.QueryEscape("YWJj")} {
		w := testSrv.GET("/api/v1/events?"+q, h)
		assert.True(t, w.Code == 400 || w.Code == 422, "%s → %d %s", q, w.Code, w.Body.String())
	}
}

func TestEvent_List_Success_RoutedToCallerSolution(t *testing.T) {
	f := newEventsFixture(t)

	page := listEvents(t, f.token, url.Values{"company_id": {f.sub.companyID}})
	types := eventTypesOf(page.Data)
	assert.Contains(t, types, "subscription.created", "event solution sendiri")
	assert.Contains(t, types, "branch.created", "event company (solution_id NULL) dari company pelanggan")
	for _, e := range page.Data {
		require.NotNil(t, e.CompanyID)
		assert.Equal(t, f.sub.companyID, *e.CompanyID)
		assert.NotEmpty(t, e.Data)
	}
	assert.Equal(t, 50, page.Meta.Limit)
	assert.Nil(t, page.Meta.NextCursor)

	// Company yang tidak berlangganan tidak terlihat.
	assert.Empty(t, listEvents(t, f.token, url.Values{"company_id": {f.otherCompany}}).Data)
	// Solution lain tidak melihat event company fixture.
	assert.Empty(t, listEvents(t, f.otherToken, url.Values{"company_id": {f.sub.companyID}}).Data)
}

func TestEvent_List_Filters(t *testing.T) {
	f := newEventsFixture(t)

	page := listEvents(t, f.token, url.Values{"company_id": {f.sub.companyID}, "event_type": {"branch.created"}})
	require.Len(t, page.Data, 1)
	assert.Equal(t, "branch.created", page.Data[0].Type)
	assert.Nil(t, page.Data[0].SolutionID)

	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	assert.Empty(t, listEvents(t, f.token, url.Values{"company_id": {f.sub.companyID}, "since": {future}}).Data)
}

func TestEvent_List_CursorPagination(t *testing.T) {
	f := newEventsFixture(t)
	all := listEvents(t, f.token, url.Values{"company_id": {f.sub.companyID}}).Data
	require.GreaterOrEqual(t, len(all), 3)

	var walked []string
	q := url.Values{"company_id": {f.sub.companyID}, "limit": {"1"}}
	for range len(all) + 1 {
		page := listEvents(t, f.token, q)
		for _, e := range page.Data {
			walked = append(walked, e.ID)
		}
		if page.Meta.NextCursor == nil {
			break
		}
		q.Set("cursor", *page.Meta.NextCursor)
	}
	want := make([]string, 0, len(all))
	for _, e := range all {
		want = append(want, e.ID)
	}
	assert.Equal(t, want, walked, "cursor menelusuri semua event tanpa duplikat")
}

// ── Dispatcher (worker) ──────────────────────────────────────────────────────

type receivedHook struct {
	headers http.Header
	body    []byte
}

// hookReceiver adalah endpoint webhook palsu; status menentukan kode balasan.
type hookReceiver struct {
	*httptest.Server
	mu       sync.Mutex
	status   int
	received []receivedHook
}

func newHookReceiver(t *testing.T, status int) *hookReceiver {
	t.Helper()
	r := &hookReceiver{status: status}
	r.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		r.mu.Lock()
		r.received = append(r.received, receivedHook{headers: req.Header.Clone(), body: body})
		status := r.status
		r.mu.Unlock()
		w.WriteHeader(status)
	}))
	t.Cleanup(r.Close)
	return r
}

func (r *hookReceiver) all() []receivedHook {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]receivedHook(nil), r.received...)
}

func (r *hookReceiver) setStatus(s int) {
	r.mu.Lock()
	r.status = s
	r.mu.Unlock()
}

type deliveryRow struct {
	status       string
	attempts     int
	responseCode *int
	nextAttempt  *time.Time
	deliveredAt  *time.Time
}

func deliveriesFor(t *testing.T, webhookID string) []deliveryRow {
	t.Helper()
	rows, err := testSrv.DB.QueryContext(context.Background(), `
		SELECT status, attempt_count, last_response_code, next_attempt_at, delivered_at
		FROM webhook_deliveries WHERE webhook_endpoint_id = $1 ORDER BY created_at`, webhookID)
	require.NoError(t, err)
	defer rows.Close()
	var out []deliveryRow
	for rows.Next() {
		var d deliveryRow
		require.NoError(t, rows.Scan(&d.status, &d.attempts, &d.responseCode, &d.nextAttempt, &d.deliveredAt))
		out = append(out, d)
	}
	require.NoError(t, rows.Err())
	return out
}

func TestWebhookDispatch_Success_SignedAndDeliveredOnce(t *testing.T) {
	f := newSubFixture(t)
	mustSubscribe(t, f.companyID, f.basicPlan)
	recv := newHookReceiver(t, 204)
	hook := mustCreateWebhook(t, f.solutionID, recv.URL+"/hooks", "branch.created")
	companyID := mustCreateBranch(t, adminHeader(), f.companyID, testhelper.UniqueCode("co"))

	d := testSrv.NewDispatcher()
	_, _, err := d.RunOnce(context.Background())
	require.NoError(t, err)

	got := recv.all()
	require.Len(t, got, 1)
	h := got[0].headers
	assert.Equal(t, "branch.created", h.Get("X-Platform-Event-Type"))
	assert.Equal(t, "application/json", h.Get("Content-Type"))
	ts, err := strconv.ParseInt(h.Get("X-Platform-Timestamp"), 10, 64)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), time.Unix(ts, 0), time.Minute)
	assert.True(t, service.Verify(hook.Secret, ts, got[0].body, h.Get("X-Platform-Signature")), "signature valid")
	assert.False(t, service.Verify("whsec_salah", ts, got[0].body, h.Get("X-Platform-Signature")))

	var body eventData
	require.NoError(t, json.Unmarshal(got[0].body, &body))
	assert.Equal(t, h.Get("X-Platform-Event-Id"), body.ID)
	assert.Equal(t, "branch.created", body.Type)
	require.NotNil(t, body.CompanyID)
	assert.Equal(t, f.companyID, *body.CompanyID)
	var payload struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(body.Data, &payload))
	assert.Equal(t, companyID, payload.ID)

	rows := deliveriesFor(t, hook.ID)
	require.Len(t, rows, 1)
	assert.Equal(t, "success", rows[0].status)
	assert.Equal(t, 1, rows[0].attempts)
	require.NotNil(t, rows[0].responseCode)
	assert.Equal(t, 204, *rows[0].responseCode)
	assert.NotNil(t, rows[0].deliveredAt)

	// Siklus berikutnya tidak membuat/mengirim ulang.
	_, _, err = d.RunOnce(context.Background())
	require.NoError(t, err)
	assert.Len(t, recv.all(), 1)
	assert.Len(t, deliveriesFor(t, hook.ID), 1)
}

func TestWebhookDispatch_RetryBackoffThenDead(t *testing.T) {
	f := newSubFixture(t)
	mustSubscribe(t, f.companyID, f.basicPlan)
	recv := newHookReceiver(t, 500)
	hook := mustCreateWebhook(t, f.solutionID, recv.URL, "branch.created")
	mustCreateBranch(t, adminHeader(), f.companyID, testhelper.UniqueCode("co"))

	now := time.Now()
	d := testSrv.NewDispatcher()
	d.SetClock(func() time.Time { return now })
	ctx := context.Background()

	_, _, err := d.RunOnce(ctx)
	require.NoError(t, err)
	rows := deliveriesFor(t, hook.ID)
	require.Len(t, rows, 1)
	assert.Equal(t, "failed", rows[0].status)
	assert.Equal(t, 1, rows[0].attempts)
	require.NotNil(t, rows[0].responseCode)
	assert.Equal(t, 500, *rows[0].responseCode)
	require.NotNil(t, rows[0].nextAttempt)
	assert.WithinDuration(t, now.Add(30*time.Second), *rows[0].nextAttempt, time.Second)

	// Belum jatuh tempo → tidak dikirim.
	_, _, err = d.RunOnce(ctx)
	require.NoError(t, err)
	assert.Len(t, recv.all(), 1)

	// Jatuh tempo → percobaan ke-2, backoff 60 detik.
	now = now.Add(31 * time.Second)
	_, _, err = d.RunOnce(ctx)
	require.NoError(t, err)
	assert.Len(t, recv.all(), 2)
	rows = deliveriesFor(t, hook.ID)
	assert.Equal(t, 2, rows[0].attempts)
	assert.WithinDuration(t, now.Add(time.Minute), *rows[0].nextAttempt, time.Second)

	// Percobaan terakhir gagal → dead, tidak dijadwalkan lagi.
	_, err = testSrv.DB.ExecContext(ctx, `UPDATE webhook_deliveries SET attempt_count = 7 WHERE webhook_endpoint_id = $1`, hook.ID)
	require.NoError(t, err)
	now = now.Add(2 * time.Minute)
	_, _, err = d.RunOnce(ctx)
	require.NoError(t, err)
	rows = deliveriesFor(t, hook.ID)
	assert.Equal(t, "dead", rows[0].status)
	assert.Equal(t, 8, rows[0].attempts)
	assert.Nil(t, rows[0].nextAttempt)

	now = now.Add(24 * time.Hour)
	recv.setStatus(200)
	_, _, err = d.RunOnce(ctx)
	require.NoError(t, err)
	assert.Len(t, recv.all(), 3, "delivery dead tidak dikirim otomatis")
}

func TestWebhookDispatch_RecoversAfterFailure(t *testing.T) {
	f := newSubFixture(t)
	mustSubscribe(t, f.companyID, f.basicPlan)
	recv := newHookReceiver(t, 503)
	hook := mustCreateWebhook(t, f.solutionID, recv.URL, "branch.created")
	mustCreateBranch(t, adminHeader(), f.companyID, testhelper.UniqueCode("co"))

	now := time.Now()
	d := testSrv.NewDispatcher()
	d.SetClock(func() time.Time { return now })
	_, _, err := d.RunOnce(context.Background())
	require.NoError(t, err)

	recv.setStatus(200)
	now = now.Add(time.Minute)
	_, _, err = d.RunOnce(context.Background())
	require.NoError(t, err)

	rows := deliveriesFor(t, hook.ID)
	require.Len(t, rows, 1)
	assert.Equal(t, "success", rows[0].status)
	assert.Equal(t, 2, rows[0].attempts)
	assert.Nil(t, rows[0].nextAttempt)
	assert.NotNil(t, rows[0].deliveredAt)
}

func TestWebhookDispatch_OnlyRoutedEvents(t *testing.T) {
	f := newSubFixture(t)
	mustSubscribe(t, f.companyID, f.basicPlan)
	recv := newHookReceiver(t, 200)

	// Solution lain yang tidak dilanggani company fixture.
	otherHook := mustCreateWebhook(t, mustCreateSolution(t, testhelper.UniqueCode("oth")), recv.URL, "branch.created")
	// Solution fixture, tetapi tidak berlangganan branch.created.
	unsubscribedHook := mustCreateWebhook(t, f.solutionID, recv.URL, "company.updated")
	// Event sebelum webhook dibuat tidak dikirim ke webhook baru.
	mustCreateBranch(t, adminHeader(), f.companyID, testhelper.UniqueCode("co"))
	lateHook := mustCreateWebhook(t, f.solutionID, recv.URL, "branch.created")

	_, _, err := testSrv.NewDispatcher().RunOnce(context.Background())
	require.NoError(t, err)

	assert.Empty(t, deliveriesFor(t, otherHook.ID), "company tidak subscribe solution")
	assert.Empty(t, deliveriesFor(t, unsubscribedHook.ID), "event type tidak dilanggani")
	assert.Empty(t, deliveriesFor(t, lateHook.ID), "event lebih lama dari webhook")
	assert.Empty(t, recv.all())
}
