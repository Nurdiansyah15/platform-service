package api_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/testhelper"
)

func permissionsPath(solutionID string) string {
	return "/api/v1/solutions/" + solutionID + "/permissions"
}
func syncPath(solutionID string) string { return permissionsPath(solutionID) + "/sync" }

type syncResult struct {
	Version     string `json:"version"`
	Changed     bool   `json:"changed"`
	Created     int    `json:"created"`
	Updated     int    `json:"updated"`
	Reactivated int    `json:"reactivated"`
	Deprecated  int    `json:"deprecated"`
	Unchanged   int    `json:"unchanged"`
}

func catalog(version string, perms ...[2]string) map[string]any {
	items := []any{}
	for _, p := range perms {
		items = append(items, map[string]any{"code": p[0], "name": p[1]})
	}
	return map[string]any{"version": version, "permissions": items}
}

func mustSync(t *testing.T, solutionID string, h map[string]string, body map[string]any) syncResult {
	t.Helper()
	w := testSrv.PUT(syncPath(solutionID), body, h)
	require.Equal(t, 200, w.Code, w.Body.String())
	var got syncResult
	testhelper.DecodeBody(t, w, &got)
	return got
}

func permissionStatuses(t *testing.T, solutionID string) map[string]string {
	t.Helper()
	rows, err := testSrv.DB.QueryContext(context.Background(), `SELECT code, status FROM permissions WHERE solution_id = $1`, solutionID)
	require.NoError(t, err)
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var c, s string
		require.NoError(t, rows.Scan(&c, &s))
		out[c] = s
	}
	return out
}

// ── Akses ────────────────────────────────────────────────────────────────────

func TestCatalogSync_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.PUT(syncPath(uuid.NewString()), catalog("v1")).Code)
}

func TestCatalogSync_Forbidden(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	body := catalog("v1", [2]string{"employee.read", "Read"})

	tok, _, _ := mustLoginNewUser(t)
	assert.Equal(t, 403, testSrv.PUT(syncPath(solID), body, testhelper.BearerHeader(tok.AccessToken)).Code, "user biasa")

	otherSol := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	foreign := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, otherSol, "catalog:sync"), ""))
	assert.Equal(t, 403, testSrv.PUT(syncPath(solID), body, foreign).Code, "client solution lain")

	noScope := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, solID, "usage:write"), ""))
	assert.Equal(t, 403, testSrv.PUT(syncPath(solID), body, noScope).Code, "tanpa catalog:sync")
}

func TestCatalogSync_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.PUT(syncPath(uuid.NewString()), catalog("v1"), adminHeader()).Code)
}

func TestCatalogSync_InvalidPayload(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	tests := map[string]map[string]any{
		"tanpa version":     {"permissions": []any{}},
		"tanpa permissions": {"version": "v1"},
		"code huruf besar":  catalog("v1", [2]string{"Employee.Read", "X"}),
		"code duplikat":     catalog("v1", [2]string{"a.read", "A"}, [2]string{"a.read", "A2"}),
		"tanpa name":        {"version": "v1", "permissions": []any{map[string]any{"code": "a.read"}}},
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, 422, testSrv.PUT(syncPath(solID), body, adminHeader()).Code)
		})
	}
}

// ── Sukses ───────────────────────────────────────────────────────────────────

func TestCatalogSync_Lifecycle(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	svc := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, solID, "catalog:sync"), ""))

	got := mustSync(t, solID, svc, catalog("2026.10.1",
		[2]string{"employee.read", "Read employee"}, [2]string{"employee.write", "Write employee"}, [2]string{"legacy:export", "Legacy"}))
	assert.Equal(t, syncResult{Version: "2026.10.1", Changed: true, Created: 3}, got)

	// Versi sama → tidak ada perubahan walau isi berbeda.
	got = mustSync(t, solID, svc, catalog("2026.10.1", [2]string{"other.perm", "X"}))
	assert.False(t, got.Changed)
	assert.Len(t, permissionStatuses(t, solID), 3)

	// Versi baru: write diubah, legacy hilang (deprecated), payroll baru.
	got = mustSync(t, solID, svc, catalog("2026.10.2",
		[2]string{"employee.read", "Read employee"}, [2]string{"employee.write", "Write employees"}, [2]string{"payroll:approve", "Approve"}))
	assert.Equal(t, syncResult{Version: "2026.10.2", Changed: true, Created: 1, Updated: 1, Deprecated: 1, Unchanged: 1}, got)
	assert.Equal(t, map[string]string{
		"employee.read": "active", "employee.write": "active", "legacy:export": "deprecated", "payroll:approve": "active",
	}, permissionStatuses(t, solID), "permission deprecated tidak dihapus")

	// Legacy kembali di katalog → diaktifkan ulang.
	got = mustSync(t, solID, adminHeader(), catalog("2026.10.3",
		[2]string{"employee.read", "Read employee"}, [2]string{"employee.write", "Write employees"},
		[2]string{"payroll:approve", "Approve"}, [2]string{"legacy:export", "Legacy"}))
	assert.Equal(t, 1, got.Reactivated)
	assert.Equal(t, "active", permissionStatuses(t, solID)["legacy:export"])

	var version string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT permission_catalog_version FROM solutions WHERE id = $1`, solID).Scan(&version))
	assert.Equal(t, "2026.10.3", version)
}

func TestCatalogSync_ConcurrentSameVersion_AppliedOnce(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	body := catalog("v1", [2]string{"a.read", "A"}, [2]string{"b.read", "B"})

	const n = 5
	var wg sync.WaitGroup
	results := make([]syncResult, n)
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w := testSrv.PUT(syncPath(solID), body, adminHeader())
			codes[i] = w.Code
			var resp struct {
				Data syncResult `json:"data"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			results[i] = resp.Data
		}(i)
	}
	wg.Wait()

	changed := 0
	for i := range results {
		assert.Equal(t, 200, codes[i])
		if results[i].Changed {
			changed++
		}
	}
	assert.Equal(t, 1, changed, "lock baris solution: hanya satu sync yang menerapkan perubahan")
	assert.Len(t, permissionStatuses(t, solID), 2)
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestPermission_List(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	mustSync(t, solID, adminHeader(), catalog("v1", [2]string{"a.read", "A"}, [2]string{"b.read", "B"}))
	mustSync(t, solID, adminHeader(), catalog("v2", [2]string{"a.read", "A"}))

	assert.Equal(t, 401, testSrv.GET(permissionsPath(solID)).Code)
	assert.Equal(t, 404, testSrv.GET(permissionsPath(uuid.NewString()), userHeader()).Code)
	assert.Equal(t, 422, testSrv.GET(permissionsPath(solID)+"?status=removed", userHeader()).Code)

	w := testSrv.GET(permissionsPath(solID)+"?status=deprecated", userHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp struct {
		Data []struct {
			Code   string `json:"code"`
			Status string `json:"status"`
		} `json:"data"`
		Meta struct {
			Total int64 `json:"total"`
		} `json:"meta"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "b.read", resp.Data[0].Code)
	assert.Equal(t, int64(1), resp.Meta.Total)
}

func TestPermission_List_InactiveSolutionHidden(t *testing.T) {
	solID := testhelper.MustSeedSolution(t, testSrv.DB, testhelper.UniqueCode("off"), "inactive")
	assert.Equal(t, 404, testSrv.GET(permissionsPath(solID), userHeader()).Code)
	assert.Equal(t, 200, testSrv.GET(permissionsPath(solID), adminHeader()).Code)
}
