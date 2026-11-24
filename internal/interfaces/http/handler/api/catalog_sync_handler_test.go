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

// ── Permission groups & scope types ──────────────────────────────────────────

func permissionGroupsPath(solutionID string) string {
	return "/api/v1/solutions/" + solutionID + "/permission-groups"
}

func scopeTypesPath(solutionID string) string {
	return "/api/v1/solutions/" + solutionID + "/scope-types"
}

// richCatalog: scope type department (parent branch), group leave & payroll, permission
// leave.approve (grup leave, scope company+department), payroll.run (grup payroll,
// company saja), misc.read (tanpa grup & scope).
func richCatalog(version string) map[string]any {
	return map[string]any{
		"version": version,
		"scope_types": []any{
			map[string]any{"code": "department", "name": "Departemen", "parent": "branch"},
		},
		"groups": []any{
			map[string]any{"code": "leave", "name": "Cuti"},
			map[string]any{"code": "payroll", "name": "Payroll", "description": "Penggajian"},
		},
		"permissions": []any{
			map[string]any{"code": "leave.approve", "name": "Setujui cuti", "group": "leave", "scopes": []string{"department", "company"}},
			map[string]any{"code": "payroll.run", "name": "Jalankan payroll", "group": "payroll", "scopes": []string{"company"}},
			map[string]any{"code": "misc.read", "name": "Lihat lain-lain"},
		},
	}
}

func TestCatalogSync_WithGroupsAndScopes_Success(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	w := testSrv.PUT(syncPath(solID), richCatalog("v1"), adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got struct {
		Created    int                   `json:"created"`
		Groups     struct{ Created int } `json:"groups"`
		ScopeTypes struct{ Created int } `json:"scope_types"`
	}
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, 3, got.Created)
	assert.Equal(t, 2, got.Groups.Created)
	assert.Equal(t, 1, got.ScopeTypes.Created)

	w = testSrv.GET(permissionsPath(solID), userHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var list []struct {
		Code  string `json:"code"`
		Group *struct {
			Code      string `json:"code"`
			Name      string `json:"name"`
			SortOrder int    `json:"sort_order"`
		} `json:"group"`
		ScopeTypes []string `json:"scope_types"`
	}
	testhelper.DecodeBody(t, w, &list)
	byCode := map[string]int{}
	for i, p := range list {
		byCode[p.Code] = i
	}
	leave := list[byCode["leave.approve"]]
	require.NotNil(t, leave.Group)
	assert.Equal(t, "Cuti", leave.Group.Name)
	assert.Equal(t, []string{"company", "department"}, leave.ScopeTypes, "unik & terurut")
	payroll := list[byCode["payroll.run"]]
	require.NotNil(t, payroll.Group)
	assert.Equal(t, 1, payroll.Group.SortOrder)
	misc := list[byCode["misc.read"]]
	assert.Nil(t, misc.Group)
	assert.Equal(t, []string{}, misc.ScopeTypes)

	// Filter group.
	w = testSrv.GET(permissionsPath(solID)+"?group=-", userHeader())
	require.Equal(t, 200, w.Code)
	testhelper.DecodeBody(t, w, &list)
	require.Len(t, list, 1)
	assert.Equal(t, "misc.read", list[0].Code)

	// Group & scope type yang hilang dari katalog → deprecated; permission lepas dari grup.
	mustSync(t, solID, adminHeader(), catalog("v2", [2]string{"leave.approve", "Setujui cuti"}))
	var groupStatus, scopeStatus string
	var groupID *string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT status FROM permission_groups WHERE solution_id = $1 AND code = 'leave'`, solID).Scan(&groupStatus))
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT status FROM solution_scope_types WHERE solution_id = $1 AND code = 'department'`, solID).Scan(&scopeStatus))
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT group_id FROM permissions WHERE solution_id = $1 AND code = 'leave.approve'`, solID).Scan(&groupID))
	assert.Equal(t, "deprecated", groupStatus)
	assert.Equal(t, "deprecated", scopeStatus)
	assert.Nil(t, groupID)
}

func TestCatalogSync_WithGroupsAndScopes_InvalidPayload(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	perm := func(extra map[string]any) []any {
		p := map[string]any{"code": "a.read", "name": "A"}
		for k, v := range extra {
			p[k] = v
		}
		return []any{p}
	}
	tests := map[string]map[string]any{
		"group tidak dikenal": {"version": "v1", "permissions": perm(map[string]any{"group": "nope"})},
		"scope tidak dikenal": {"version": "v1", "permissions": perm(map[string]any{"scopes": []string{"project"}})},
		"scope type reserved": {"version": "v1", "permissions": perm(nil),
			"scope_types": []any{map[string]any{"code": "branch", "name": "Cabang"}}},
		"scope type ganda": {"version": "v1", "permissions": perm(nil),
			"scope_types": []any{map[string]any{"code": "dept", "name": "A"}, map[string]any{"code": "dept", "name": "B"}}},
		"group ganda": {"version": "v1", "permissions": perm(nil),
			"groups": []any{map[string]any{"code": "g", "name": "A"}, map[string]any{"code": "g", "name": "B"}}},
		"parent tidak valid": {"version": "v1", "permissions": perm(nil),
			"scope_types": []any{map[string]any{"code": "dept", "name": "A", "parent": "region"}}},
		"group tanpa nama": {"version": "v1", "permissions": perm(nil), "groups": []any{map[string]any{"code": "g"}}},
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			w := testSrv.PUT(syncPath(solID), body, adminHeader())
			assert.Equal(t, 422, w.Code, w.Body.String())
		})
	}
}

func TestCatalogSync_LegacyPayload_Success(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	got := mustSync(t, solID, adminHeader(), catalog("v1", [2]string{"a.read", "A"}))
	assert.Equal(t, 1, got.Created)

	w := testSrv.GET(scopeTypesPath(solID), userHeader())
	require.Equal(t, 200, w.Code)
	var scopes []struct {
		Code string `json:"code"`
	}
	testhelper.DecodeBody(t, w, &scopes)
	require.Len(t, scopes, 2, "hanya built-in")
}

func TestPermissionGroups_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(permissionGroupsPath(uuid.NewString())).Code)
}

func TestPermissionGroups_List_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.GET(permissionGroupsPath(uuid.NewString()), userHeader()).Code)
}

func TestPermissionGroups_List_InvalidQuery(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	assert.Equal(t, 422, testSrv.GET(permissionGroupsPath(solID)+"?status=gone", userHeader()).Code)
}

func TestPermissionGroups_List_Success(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	mustSync(t, solID, adminHeader(), richCatalog("v1"))

	w := testSrv.GET(permissionGroupsPath(solID), userHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got []struct {
		Code        string  `json:"code"`
		Name        string  `json:"name"`
		Description *string `json:"description"`
		SortOrder   int     `json:"sort_order"`
		Status      string  `json:"status"`
	}
	testhelper.DecodeBody(t, w, &got)
	require.Len(t, got, 2)
	assert.Equal(t, "leave", got[0].Code)
	assert.Equal(t, "payroll", got[1].Code)
	require.NotNil(t, got[1].Description)
	assert.Equal(t, "Penggajian", *got[1].Description)
	assert.Equal(t, "active", got[0].Status)
}

func TestScopeTypes_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(scopeTypesPath(uuid.NewString())).Code)
}

func TestScopeTypes_List_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.GET(scopeTypesPath(uuid.NewString()), userHeader()).Code)
}

func TestScopeTypes_List_Success(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	mustSync(t, solID, adminHeader(), richCatalog("v1"))

	w := testSrv.GET(scopeTypesPath(solID), userHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got []struct {
		Code    string `json:"code"`
		Name    string `json:"name"`
		Parent  string `json:"parent"`
		Builtin bool   `json:"builtin"`
	}
	testhelper.DecodeBody(t, w, &got)
	require.Len(t, got, 3)
	assert.Equal(t, "company", got[0].Code)
	assert.True(t, got[0].Builtin)
	assert.Equal(t, "branch", got[1].Code)
	assert.Equal(t, "department", got[2].Code)
	assert.Equal(t, "Departemen", got[2].Name)
	assert.Equal(t, "branch", got[2].Parent)
	assert.False(t, got[2].Builtin)
}

// ── Permission manual (CRUD platform) ────────────────────────────────────────

func permissionPath(id string) string { return "/api/v1/permissions/" + id }

type permissionBody struct {
	ID     string `json:"id"`
	Code   string `json:"code"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Source string `json:"source"`
	Group  *struct {
		Code string `json:"code"`
	} `json:"group"`
	ScopeTypes  []string `json:"scope_types"`
	Description *string  `json:"description"`
}

func mustCreatePermission(t *testing.T, solutionID string, body map[string]any) permissionBody {
	t.Helper()
	w := testSrv.POST(permissionsPath(solutionID), body, adminHeader())
	require.Equal(t, 201, w.Code, "create permission: %s", w.Body.String())
	var got permissionBody
	testhelper.DecodeBody(t, w, &got)
	return got
}

func TestPermission_Create_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(permissionsPath(uuid.NewString()), map[string]any{"code": "a.read", "name": "A"}).Code)
}

func TestPermission_Create_Forbidden(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	body := map[string]any{"code": "a.read", "name": "A"}
	assert.Equal(t, 403, testSrv.POST(permissionsPath(solID), body, userHeader()).Code)
	svc := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, solID, "catalog:sync"), ""))
	assert.Equal(t, 403, testSrv.POST(permissionsPath(solID), body, svc).Code)
}

func TestPermission_Create_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.POST(permissionsPath(uuid.NewString()), map[string]any{"code": "a.read", "name": "A"}, adminHeader()).Code)
}

func TestPermission_Create_InvalidPayload(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	mustSync(t, solID, adminHeader(), richCatalog("v1"))
	tests := map[string]map[string]any{
		"tanpa name":          {"code": "a.read"},
		"code tidak valid":    {"code": "A Read", "name": "A"},
		"group tidak dikenal": {"code": "a.read", "name": "A", "group": "nope"},
		"scope tidak dikenal": {"code": "a.read", "name": "A", "scopes": []string{"project"}},
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			w := testSrv.POST(permissionsPath(solID), body, adminHeader())
			assert.True(t, w.Code == 400 || w.Code == 422, "%d %s", w.Code, w.Body.String())
		})
	}
}

func TestPermission_Create_Duplicate(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	mustSync(t, solID, adminHeader(), catalog("v1", [2]string{"a.read", "A"}))
	assert.Equal(t, 409, testSrv.POST(permissionsPath(solID), map[string]any{"code": "a.read", "name": "A"}, adminHeader()).Code, "bentrok code sync")
	mustCreatePermission(t, solID, map[string]any{"code": "b.read", "name": "B"})
	assert.Equal(t, 409, testSrv.POST(permissionsPath(solID), map[string]any{"code": "b.read", "name": "B"}, adminHeader()).Code, "bentrok code manual")
}

func TestPermission_Create_Success(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	mustSync(t, solID, adminHeader(), richCatalog("v1"))
	got := mustCreatePermission(t, solID, map[string]any{
		"code": "report.export", "name": "Export laporan", "description": "CSV", "group": "leave", "scopes": []string{"department"},
	})
	assert.Equal(t, "manual", got.Source)
	assert.Equal(t, "active", got.Status)
	require.NotNil(t, got.Group)
	assert.Equal(t, "leave", got.Group.Code)
	assert.Equal(t, []string{"department"}, got.ScopeTypes)

	// Muncul di list & filter source; bisa dipasang di role.
	w := testSrv.GET(permissionsPath(solID)+"?source=manual", userHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var list []permissionBody
	testhelper.DecodeBody(t, w, &list)
	require.Len(t, list, 1)
	assert.Equal(t, "report.export", list[0].Code)

	roleID := mustCreateRole(t, solutionRolesPath(solID), "exporter", adminHeader())
	assert.Equal(t, 200, testSrv.PUT(rolePermissionsPath(roleID), map[string]any{"permission_codes": []string{"report.export"}}, adminHeader()).Code)
}

func TestPermission_Get(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	p := mustCreatePermission(t, solID, map[string]any{"code": "a.read", "name": "A"})

	assert.Equal(t, 401, testSrv.GET(permissionPath(p.ID)).Code)
	assert.Equal(t, 404, testSrv.GET(permissionPath(uuid.NewString()), userHeader()).Code)
	assert.Equal(t, 422, testSrv.GET(permissionPath("not-uuid"), userHeader()).Code)

	w := testSrv.GET(permissionPath(p.ID), userHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got permissionBody
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "a.read", got.Code)
	assert.Equal(t, "manual", got.Source)
}

func TestPermission_Update_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.PATCH(permissionPath(uuid.NewString()), map[string]any{"name": "X"}).Code)
}

func TestPermission_Update_Forbidden(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	p := mustCreatePermission(t, solID, map[string]any{"code": "a.read", "name": "A"})
	assert.Equal(t, 403, testSrv.PATCH(permissionPath(p.ID), map[string]any{"name": "X"}, userHeader()).Code)
}

func TestPermission_Update_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.PATCH(permissionPath(uuid.NewString()), map[string]any{"name": "X"}, adminHeader()).Code)
}

func TestPermission_Update_InvalidPayload(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	p := mustCreatePermission(t, solID, map[string]any{"code": "a.read", "name": "A"})
	assert.Equal(t, 422, testSrv.PATCH(permissionPath(p.ID), map[string]any{"status": "removed"}, adminHeader()).Code)
	assert.Equal(t, 422, testSrv.PATCH(permissionPath(p.ID), map[string]any{"name": "  "}, adminHeader()).Code)
	assert.Equal(t, 422, testSrv.PATCH(permissionPath(p.ID), map[string]any{"group": "nope"}, adminHeader()).Code)
}

func TestPermission_Update_SyncManaged_Conflict(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	mustSync(t, solID, adminHeader(), catalog("v1", [2]string{"a.read", "A"}))
	var id string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT id FROM permissions WHERE solution_id = $1 AND code = 'a.read'`, solID).Scan(&id))
	assert.Equal(t, 409, testSrv.PATCH(permissionPath(id), map[string]any{"name": "X"}, adminHeader()).Code)
}

func TestPermission_Update_Success(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	mustSync(t, solID, adminHeader(), richCatalog("v1"))
	p := mustCreatePermission(t, solID, map[string]any{"code": "x.read", "name": "X", "description": "d", "group": "leave"})

	w := testSrv.PATCH(permissionPath(p.ID), map[string]any{"name": "Lihat X", "description": "", "group": "", "scopes": []string{"company"}}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got permissionBody
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "Lihat X", got.Name)
	assert.Nil(t, got.Description)
	assert.Nil(t, got.Group)
	assert.Equal(t, []string{"company"}, got.ScopeTypes)
	assert.Equal(t, "x.read", got.Code)

	// "Hapus" = deprecated: tidak bisa ditambahkan ke role; aktifkan lagi bisa.
	w = testSrv.PATCH(permissionPath(p.ID), map[string]any{"status": "deprecated"}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "deprecated", got.Status)
	roleID := mustCreateRole(t, solutionRolesPath(solID), "xr", adminHeader())
	assert.Equal(t, 422, testSrv.PUT(rolePermissionsPath(roleID), map[string]any{"permission_codes": []string{"x.read"}}, adminHeader()).Code)

	w = testSrv.PATCH(permissionPath(p.ID), map[string]any{"status": "active"}, adminHeader())
	require.Equal(t, 200, w.Code)
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "active", got.Status)
}

func TestCatalogSync_ManualPermission(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	mustCreatePermission(t, solID, map[string]any{"code": "custom.export", "name": "Export"})

	// Sync tidak men-deprecate permission manual.
	mustSync(t, solID, adminHeader(), catalog("v1", [2]string{"a.read", "A"}))
	assert.Equal(t, map[string]string{"a.read": "active", "custom.export": "active"}, permissionStatuses(t, solID))

	// Sync yang memuat code manual ditolak utuh.
	w := testSrv.PUT(syncPath(solID), catalog("v2", [2]string{"a.read", "A"}, [2]string{"custom.export", "X"}), adminHeader())
	assert.Equal(t, 409, w.Code, w.Body.String())
	var version *string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT permission_catalog_version FROM solutions WHERE id = $1`, solID).Scan(&version))
	require.NotNil(t, version)
	assert.Equal(t, "v1", *version)
}

// ── Permission group manual (CRUD platform) ─────────────────────────────────

func permissionGroupPath(id string) string { return "/api/v1/permission-groups/" + id }

type groupBody struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	SortOrder   int     `json:"sort_order"`
	Status      string  `json:"status"`
	Source      string  `json:"source"`
}

func mustCreateGroup(t *testing.T, solutionID string, body map[string]any) groupBody {
	t.Helper()
	w := testSrv.POST(permissionGroupsPath(solutionID), body, adminHeader())
	require.Equal(t, 201, w.Code, "create group: %s", w.Body.String())
	var got groupBody
	testhelper.DecodeBody(t, w, &got)
	return got
}

func TestPermissionGroup_Create_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(permissionGroupsPath(uuid.NewString()), map[string]any{"code": "g", "name": "G"}).Code)
}

func TestPermissionGroup_Create_Forbidden(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	assert.Equal(t, 403, testSrv.POST(permissionGroupsPath(solID), map[string]any{"code": "g", "name": "G"}, userHeader()).Code)
}

func TestPermissionGroup_Create_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.POST(permissionGroupsPath(uuid.NewString()), map[string]any{"code": "g", "name": "G"}, adminHeader()).Code)
}

func TestPermissionGroup_Create_InvalidPayload(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	for name, body := range map[string]map[string]any{
		"tanpa name":       {"code": "g"},
		"code tidak valid": {"code": "Grup A", "name": "G"},
		"sort negatif":     {"code": "g", "name": "G", "sort_order": -1},
	} {
		t.Run(name, func(t *testing.T) {
			w := testSrv.POST(permissionGroupsPath(solID), body, adminHeader())
			assert.True(t, w.Code == 400 || w.Code == 422, "%d %s", w.Code, w.Body.String())
		})
	}
}

func TestPermissionGroup_Create_Duplicate(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	mustSync(t, solID, adminHeader(), richCatalog("v1"))
	assert.Equal(t, 409, testSrv.POST(permissionGroupsPath(solID), map[string]any{"code": "leave", "name": "X"}, adminHeader()).Code, "bentrok group sync")
	mustCreateGroup(t, solID, map[string]any{"code": "report", "name": "Laporan"})
	assert.Equal(t, 409, testSrv.POST(permissionGroupsPath(solID), map[string]any{"code": "report", "name": "X"}, adminHeader()).Code, "bentrok group manual")
}

func TestPermissionGroup_Create_Success(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	mustSync(t, solID, adminHeader(), richCatalog("v1")) // leave=0, payroll=1
	got := mustCreateGroup(t, solID, map[string]any{"code": "report", "name": "Laporan", "description": "Export"})
	assert.Equal(t, "manual", got.Source)
	assert.Equal(t, "active", got.Status)
	assert.Equal(t, 2, got.SortOrder, "default setelah group terakhir")

	// Bisa dipakai permission manual.
	p := mustCreatePermission(t, solID, map[string]any{"code": "report.export", "name": "Export", "group": "report"})
	require.NotNil(t, p.Group)
	assert.Equal(t, "report", p.Group.Code)

	// Sync berikutnya tidak men-deprecate group manual; sync yang memuat code-nya ditolak.
	mustSync(t, solID, adminHeader(), catalog("v2", [2]string{"a.read", "A"}))
	w := testSrv.GET(permissionGroupPath(got.ID), userHeader())
	require.Equal(t, 200, w.Code)
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "active", got.Status)
	w = testSrv.PUT(syncPath(solID), map[string]any{
		"version": "v3", "groups": []any{map[string]any{"code": "report", "name": "R"}}, "permissions": []any{},
	}, adminHeader())
	assert.Equal(t, 409, w.Code, w.Body.String())
}

func TestPermissionGroup_Get(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	g := mustCreateGroup(t, solID, map[string]any{"code": "report", "name": "Laporan"})
	assert.Equal(t, 401, testSrv.GET(permissionGroupPath(g.ID)).Code)
	assert.Equal(t, 404, testSrv.GET(permissionGroupPath(uuid.NewString()), userHeader()).Code)
	w := testSrv.GET(permissionGroupPath(g.ID), userHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got groupBody
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "report", got.Code)
}

func TestPermissionGroup_Update_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.PATCH(permissionGroupPath(uuid.NewString()), map[string]any{"name": "X"}).Code)
}

func TestPermissionGroup_Update_Forbidden(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	g := mustCreateGroup(t, solID, map[string]any{"code": "report", "name": "Laporan"})
	assert.Equal(t, 403, testSrv.PATCH(permissionGroupPath(g.ID), map[string]any{"name": "X"}, userHeader()).Code)
}

func TestPermissionGroup_Update_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.PATCH(permissionGroupPath(uuid.NewString()), map[string]any{"name": "X"}, adminHeader()).Code)
}

func TestPermissionGroup_Update_InvalidPayload(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	g := mustCreateGroup(t, solID, map[string]any{"code": "report", "name": "Laporan"})
	assert.Equal(t, 422, testSrv.PATCH(permissionGroupPath(g.ID), map[string]any{"status": "gone"}, adminHeader()).Code)
	assert.Equal(t, 422, testSrv.PATCH(permissionGroupPath(g.ID), map[string]any{"name": " "}, adminHeader()).Code)
}

func TestPermissionGroup_Update_SyncManaged_Conflict(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	mustSync(t, solID, adminHeader(), richCatalog("v1"))
	var id string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT id FROM permission_groups WHERE solution_id = $1 AND code = 'leave'`, solID).Scan(&id))
	assert.Equal(t, 409, testSrv.PATCH(permissionGroupPath(id), map[string]any{"name": "X"}, adminHeader()).Code)
}

func TestPermissionGroup_Update_Success(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	g := mustCreateGroup(t, solID, map[string]any{"code": "report", "name": "Laporan", "description": "d"})

	w := testSrv.PATCH(permissionGroupPath(g.ID), map[string]any{"name": "Laporan & export", "description": "", "sort_order": 9}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got groupBody
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "Laporan & export", got.Name)
	assert.Nil(t, got.Description)
	assert.Equal(t, 9, got.SortOrder)

	w = testSrv.PATCH(permissionGroupPath(g.ID), map[string]any{"status": "deprecated"}, adminHeader())
	require.Equal(t, 200, w.Code)
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "deprecated", got.Status)

	w = testSrv.GET(permissionGroupsPath(solID)+"?status=active", userHeader())
	require.Equal(t, 200, w.Code)
	var list []groupBody
	testhelper.DecodeBody(t, w, &list)
	assert.Empty(t, list)
}

// ── Scope type manual (CRUD platform) ────────────────────────────────────────

func scopeTypePath(id string) string { return "/api/v1/scope-types/" + id }

type scopeTypeBody struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Parent      string  `json:"parent"`
	SortOrder   int     `json:"sort_order"`
	Status      string  `json:"status"`
	Source      string  `json:"source"`
	Builtin     bool    `json:"builtin"`
}

func mustCreateScopeType(t *testing.T, solutionID string, body map[string]any) scopeTypeBody {
	t.Helper()
	w := testSrv.POST(scopeTypesPath(solutionID), body, adminHeader())
	require.Equal(t, 201, w.Code, "create scope type: %s", w.Body.String())
	var got scopeTypeBody
	testhelper.DecodeBody(t, w, &got)
	return got
}

func TestScopeType_Create_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(scopeTypesPath(uuid.NewString()), map[string]any{"code": "project", "name": "Proyek"}).Code)
}

func TestScopeType_Create_Forbidden(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	assert.Equal(t, 403, testSrv.POST(scopeTypesPath(solID), map[string]any{"code": "project", "name": "Proyek"}, userHeader()).Code)
}

func TestScopeType_Create_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.POST(scopeTypesPath(uuid.NewString()), map[string]any{"code": "project", "name": "Proyek"}, adminHeader()).Code)
}

func TestScopeType_Create_InvalidPayload(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	for name, body := range map[string]map[string]any{
		"tanpa name":         {"code": "project"},
		"code reserved":      {"code": "branch", "name": "Cabang"},
		"code tidak valid":   {"code": "Pro-ject", "name": "P"},
		"parent tidak valid": {"code": "project", "name": "P", "parent": "region"},
	} {
		t.Run(name, func(t *testing.T) {
			w := testSrv.POST(scopeTypesPath(solID), body, adminHeader())
			assert.True(t, w.Code == 400 || w.Code == 422, "%d %s", w.Code, w.Body.String())
		})
	}
}

func TestScopeType_Create_Duplicate(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	mustSync(t, solID, adminHeader(), richCatalog("v1"))
	assert.Equal(t, 409, testSrv.POST(scopeTypesPath(solID), map[string]any{"code": "department", "name": "X"}, adminHeader()).Code, "bentrok sync")
	mustCreateScopeType(t, solID, map[string]any{"code": "project", "name": "Proyek"})
	assert.Equal(t, 409, testSrv.POST(scopeTypesPath(solID), map[string]any{"code": "project", "name": "X"}, adminHeader()).Code, "bentrok manual")
}

func TestScopeType_Create_Success(t *testing.T) {
	f := newAuthzFixture(t)
	st := mustCreateScopeType(t, f.solutionID, map[string]any{"code": "project", "name": "Proyek", "description": "Proyek klien"})
	assert.Equal(t, "manual", st.Source)
	assert.Equal(t, "company", st.Parent, "default parent company")
	assert.Equal(t, "active", st.Status)

	// Dipakai assignment & permission manual; tidak di-deprecate oleh sync berikutnya.
	w := testSrv.POST(assignPath(f.companyID, f.staffID), map[string]any{"role_id": f.hrAdmin, "scope_type": "project", "scope_id": "P1"}, f.ownerH)
	require.Equal(t, 201, w.Code, w.Body.String())
	mustCreatePermission(t, f.solutionID, map[string]any{"code": "project.close", "name": "Tutup proyek", "scopes": []string{"project"}})
	mustSync(t, f.solutionID, adminHeader(), catalog("v-after-manual", [2]string{"employee.read", "R"}, [2]string{"employee.write", "W"}, [2]string{"attendance.edit", "A"}))
	assert.True(t, f.serviceCheck(t, "employee.read", "project", "P1").Allowed)

	// Sync yang memuat code-nya ditolak.
	w = testSrv.PUT(syncPath(f.solutionID), map[string]any{
		"version": "v-conflict", "scope_types": []any{map[string]any{"code": "project", "name": "P"}}, "permissions": []any{},
	}, adminHeader())
	assert.Equal(t, 409, w.Code, w.Body.String())
}

func TestScopeType_Get(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	st := mustCreateScopeType(t, solID, map[string]any{"code": "project", "name": "Proyek"})
	assert.Equal(t, 401, testSrv.GET(scopeTypePath(st.ID)).Code)
	assert.Equal(t, 404, testSrv.GET(scopeTypePath(uuid.NewString()), userHeader()).Code)
	w := testSrv.GET(scopeTypePath(st.ID), userHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got scopeTypeBody
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "project", got.Code)

	// Muncul di list setelah built-in.
	w = testSrv.GET(scopeTypesPath(solID), userHeader())
	require.Equal(t, 200, w.Code)
	var list []scopeTypeBody
	testhelper.DecodeBody(t, w, &list)
	require.Len(t, list, 3)
	assert.Equal(t, "builtin", list[0].Source)
	assert.Equal(t, "manual", list[2].Source)
}

func TestScopeType_Update_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.PATCH(scopeTypePath(uuid.NewString()), map[string]any{"name": "X"}).Code)
}

func TestScopeType_Update_Forbidden(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	st := mustCreateScopeType(t, solID, map[string]any{"code": "project", "name": "Proyek"})
	assert.Equal(t, 403, testSrv.PATCH(scopeTypePath(st.ID), map[string]any{"name": "X"}, userHeader()).Code)
}

func TestScopeType_Update_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.PATCH(scopeTypePath(uuid.NewString()), map[string]any{"name": "X"}, adminHeader()).Code)
}

func TestScopeType_Update_InvalidPayload(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	st := mustCreateScopeType(t, solID, map[string]any{"code": "project", "name": "Proyek"})
	assert.Equal(t, 422, testSrv.PATCH(scopeTypePath(st.ID), map[string]any{"parent": "region"}, adminHeader()).Code)
	assert.Equal(t, 422, testSrv.PATCH(scopeTypePath(st.ID), map[string]any{"name": " "}, adminHeader()).Code)
}

func TestScopeType_Update_SyncManaged_Conflict(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cs"))
	mustSync(t, solID, adminHeader(), richCatalog("v1"))
	var id string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT id FROM solution_scope_types WHERE solution_id = $1 AND code = 'department'`, solID).Scan(&id))
	assert.Equal(t, 409, testSrv.PATCH(scopeTypePath(id), map[string]any{"name": "X"}, adminHeader()).Code)
}

func TestScopeType_Update_Success(t *testing.T) {
	f := newAuthzFixture(t)
	st := mustCreateScopeType(t, f.solutionID, map[string]any{"code": "project", "name": "Proyek", "description": "d"})

	w := testSrv.PATCH(scopeTypePath(st.ID), map[string]any{"name": "Proyek klien", "description": "", "parent": "branch", "sort_order": 7}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got scopeTypeBody
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "Proyek klien", got.Name)
	assert.Nil(t, got.Description)
	assert.Equal(t, "branch", got.Parent)
	assert.Equal(t, 7, got.SortOrder)

	// Deprecated: assignment baru ditolak; aktifkan lagi bisa.
	w = testSrv.PATCH(scopeTypePath(st.ID), map[string]any{"status": "deprecated"}, adminHeader())
	require.Equal(t, 200, w.Code)
	w = testSrv.POST(assignPath(f.companyID, f.staffID), map[string]any{"role_id": f.hrAdmin, "scope_type": "project", "scope_id": "P1"}, f.ownerH)
	assert.Equal(t, 422, w.Code, w.Body.String())
	w = testSrv.PATCH(scopeTypePath(st.ID), map[string]any{"status": "active"}, adminHeader())
	require.Equal(t, 200, w.Code)
	w = testSrv.POST(assignPath(f.companyID, f.staffID), map[string]any{"role_id": f.hrAdmin, "scope_type": "project", "scope_id": "P1"}, f.ownerH)
	assert.Equal(t, 201, w.Code, w.Body.String())
}
