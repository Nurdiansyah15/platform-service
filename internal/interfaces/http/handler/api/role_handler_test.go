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

func solutionRolesPath(solutionID string) string { return "/api/v1/solutions/" + solutionID + "/roles" }
func rolePermissionsPath(roleID string) string   { return "/api/v1/roles/" + roleID + "/permissions" }
func companyRolesPath(companyID, solutionID string) string {
	return "/api/v1/companies/" + companyID + "/solutions/" + solutionID + "/roles"
}
func companyRolePermissionsPath(companyID, roleID string) string {
	return "/api/v1/companies/" + companyID + "/roles/" + roleID + "/permissions"
}

type roleData struct {
	ID        string  `json:"id"`
	CompanyID *string `json:"company_id"`
	Code      string  `json:"code"`
	IsSystem  bool    `json:"is_system"`
	Scope     string  `json:"scope"`
}

// mustCreateRole membuat role lewat path tertentu dan mengembalikan ID-nya.
func mustCreateRole(t *testing.T, path, code string, h map[string]string) string {
	t.Helper()
	w := testSrv.POST(path, map[string]any{"code": code, "name": "Role " + code}, h)
	require.Equal(t, 201, w.Code, "mustCreateRole body=%s", w.Body.String())
	var got roleData
	testhelper.DecodeBody(t, w, &got)
	return got.ID
}

func permissionCodesOf(t *testing.T, body []byte) []string {
	t.Helper()
	var resp struct {
		Data []struct {
			Code string `json:"code"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &resp), string(body))
	out := []string{}
	for _, p := range resp.Data {
		out = append(out, p.Code)
	}
	return out
}

func roleCodes(t *testing.T, body []byte) []string {
	t.Helper()
	var resp struct {
		Data []roleData `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &resp), string(body))
	out := []string{}
	for _, r := range resp.Data {
		out = append(out, r.Code)
	}
	return out
}

// roleSolution: solution dengan katalog employee.read / employee.write aktif dan legacy.export deprecated.
func roleSolution(t *testing.T) string {
	t.Helper()
	solID := mustCreateSolution(t, testhelper.UniqueCode("rl"))
	mustSync(t, solID, adminHeader(), catalog("v1", [2]string{"employee.read", "R"}, [2]string{"employee.write", "W"}, [2]string{"legacy.export", "L"}))
	mustSync(t, solID, adminHeader(), catalog("v2", [2]string{"employee.read", "R"}, [2]string{"employee.write", "W"}))
	return solID
}

// ── Role global: create & list ───────────────────────────────────────────────

func TestRole_CreateGlobal_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(solutionRolesPath(uuid.NewString()), map[string]any{"code": "x", "name": "X"}).Code)
}

func TestRole_CreateGlobal_Forbidden_NonAdmin(t *testing.T) {
	solID := roleSolution(t)
	assert.Equal(t, 403, testSrv.POST(solutionRolesPath(solID), map[string]any{"code": "x", "name": "X"}, userHeader()).Code)
}

func TestRole_CreateGlobal_Invalid(t *testing.T) {
	solID := roleSolution(t)
	assert.Equal(t, 422, testSrv.POST(solutionRolesPath(solID), map[string]any{"code": "hr admin", "name": "X"}, adminHeader()).Code)
	assert.Equal(t, 422, testSrv.POST(solutionRolesPath(solID), map[string]any{"code": "hr"}, adminHeader()).Code)
	assert.Equal(t, 404, testSrv.POST(solutionRolesPath(uuid.NewString()), map[string]any{"code": "hr", "name": "X"}, adminHeader()).Code)
}

func TestRole_CreateGlobal_Duplicate(t *testing.T) {
	solID := roleSolution(t)
	mustCreateRole(t, solutionRolesPath(solID), "hr_admin", adminHeader())
	assert.Equal(t, 409, testSrv.POST(solutionRolesPath(solID), map[string]any{"code": "HR_ADMIN", "name": "X"}, adminHeader()).Code)
}

func TestRole_CreateGlobal_Success_AndList(t *testing.T) {
	solID := roleSolution(t)
	w := testSrv.POST(solutionRolesPath(solID), map[string]any{"code": "hr_admin", "name": "HR Admin"}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())
	var got roleData
	testhelper.DecodeBody(t, w, &got)
	assert.Nil(t, got.CompanyID)
	assert.Equal(t, "global", got.Scope)

	var events int
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM events WHERE event_type = 'role.created' AND solution_id = $1 AND company_id IS NULL`, solID).Scan(&events))
	assert.Equal(t, 1, events)

	w = testSrv.GET(solutionRolesPath(solID), userHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, []string{"hr_admin"}, roleCodes(t, w.Body.Bytes()))
	assert.Equal(t, 401, testSrv.GET(solutionRolesPath(solID)).Code)
}

// ── Role global: permissions ─────────────────────────────────────────────────

func TestRole_ReplaceGlobalPermissions(t *testing.T) {
	solID := roleSolution(t)
	roleID := mustCreateRole(t, solutionRolesPath(solID), "hr_admin", adminHeader())
	path := rolePermissionsPath(roleID)

	assert.Equal(t, 401, testSrv.PUT(path, map[string]any{"permission_codes": []string{}}).Code)
	assert.Equal(t, 403, testSrv.PUT(path, map[string]any{"permission_codes": []string{}}, userHeader()).Code)
	assert.Equal(t, 404, testSrv.PUT(rolePermissionsPath(uuid.NewString()), map[string]any{"permission_codes": []string{}}, adminHeader()).Code)

	for name, codes := range map[string][]string{
		"tidak dikenal": {"payroll.run"}, "duplikat": {"employee.read", "employee.read"}, "deprecated baru": {"legacy.export"},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, 422, testSrv.PUT(path, map[string]any{"permission_codes": codes}, adminHeader()).Code)
		})
	}

	w := testSrv.PUT(path, map[string]any{"permission_codes": []string{"employee.write", "employee.read"}}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, []string{"employee.read", "employee.write"}, permissionCodesOf(t, w.Body.Bytes()))
	assert.Equal(t, 1, countEventsBySolution(t, "role.permissions_changed", solID))

	w = testSrv.GET(path, userHeader())
	require.Equal(t, 200, w.Code)
	assert.Equal(t, []string{"employee.read", "employee.write"}, permissionCodesOf(t, w.Body.Bytes()))
}

func TestRole_DeprecatedPermissionAlreadyAttachedIsKept(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("rl"))
	mustSync(t, solID, adminHeader(), catalog("v1", [2]string{"employee.read", "R"}, [2]string{"legacy.export", "L"}))
	roleID := mustCreateRole(t, solutionRolesPath(solID), "auditor", adminHeader())
	require.Equal(t, 200, testSrv.PUT(rolePermissionsPath(roleID), map[string]any{"permission_codes": []string{"employee.read", "legacy.export"}}, adminHeader()).Code)

	// legacy.export kemudian di-deprecate oleh catalog sync — role lama tidak rusak.
	mustSync(t, solID, adminHeader(), catalog("v2", [2]string{"employee.read", "R"}))
	w := testSrv.PUT(rolePermissionsPath(roleID), map[string]any{"permission_codes": []string{"legacy.export"}}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, []string{"legacy.export"}, permissionCodesOf(t, w.Body.Bytes()))
}

func TestRole_SystemRoleImmutable(t *testing.T) {
	solID := roleSolution(t)
	roleID := mustCreateRole(t, solutionRolesPath(solID), "owner", adminHeader())
	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE roles SET is_system = TRUE WHERE id = $1`, roleID)
	require.NoError(t, err)

	w := testSrv.PUT(rolePermissionsPath(roleID), map[string]any{"permission_codes": []string{"employee.read"}}, adminHeader())
	assert.Equal(t, 403, w.Code, w.Body.String())
}

// ── Role custom company ───────────────────────────────────────────────────────

// companyRoleFixture: owner company (token asli ber-tid) yang subscribe solution berkatalog permission.
func companyRoleFixture(t *testing.T) (h map[string]string, companyID, solutionID string) {
	t.Helper()
	h, _, companyID = mustCompanyOwner(t)
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT solution_id FROM subscriptions WHERE company_id = $1`, companyID).Scan(&solutionID))
	mustSync(t, solutionID, adminHeader(), catalog("v1", [2]string{"employee.read", "R"}, [2]string{"employee.write", "W"}))
	return h, companyID, solutionID
}

func TestCompanyRole_Create(t *testing.T) {
	h, companyID, solID := companyRoleFixture(t)
	path := companyRolesPath(companyID, solID)

	assert.Equal(t, 401, testSrv.POST(path, map[string]any{"code": "x", "name": "X"}).Code)
	strangerH, _, _ := mustCompanyOwner(t)
	assert.Equal(t, 403, testSrv.POST(path, map[string]any{"code": "x", "name": "X"}, strangerH).Code, "bukan member")
	assert.Equal(t, 422, testSrv.POST(path, map[string]any{"code": "x"}, h).Code)

	w := testSrv.POST(path, map[string]any{"code": "supervisor", "name": "Supervisor"}, h)
	require.Equal(t, 201, w.Code, w.Body.String())
	var got roleData
	testhelper.DecodeBody(t, w, &got)
	require.NotNil(t, got.CompanyID)
	assert.Equal(t, companyID, *got.CompanyID)
	assert.Equal(t, "company", got.Scope)
	assert.Equal(t, 1, countEvents(t, "role.created", companyID))

	assert.Equal(t, 409, testSrv.POST(path, map[string]any{"code": "supervisor", "name": "Lagi"}, h).Code)

	// Code yang sama boleh di company lain dan sebagai role global.
	mustCreateRole(t, solutionRolesPath(solID), "supervisor", adminHeader())
}

// GATE-SUBSCRIPTION (nonaktif sementara): custom role boleh dibuat walau company belum subscribe.
// func TestCompanyRole_Create_CompanyNotSubscribed(t *testing.T) {
// 	solID := roleSolution(t)
// 	companyID := mustCreateCompany(t, adminHeader())
// 	w := testSrv.POST(companyRolesPath(companyID, solID), map[string]any{"code": "x", "name": "X"}, adminHeader())
// 	assert.Equal(t, 403, w.Code, w.Body.String())
// }

func TestCompanyRole_Create_WithoutSubscription_Success(t *testing.T) {
	solID := roleSolution(t)
	companyID := mustCreateCompany(t, adminHeader())
	w := testSrv.POST(companyRolesPath(companyID, solID), map[string]any{"code": "x", "name": "X"}, adminHeader())
	assert.Equal(t, 201, w.Code, w.Body.String())
}

func TestCompanyRole_Create_ServiceToken(t *testing.T) {
	_, companyID, solID := companyRoleFixture(t)
	svc := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, solID, "roles:write"), ""))
	mustCreateRole(t, companyRolesPath(companyID, solID), "from_service", svc)

	other := roleSolution(t)
	foreign := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, other, "roles:write"), ""))
	assert.Equal(t, 403, testSrv.POST(companyRolesPath(companyID, solID), map[string]any{"code": "y", "name": "Y"}, foreign).Code)
}

func TestCompanyRole_List_GlobalPlusOwnCustom(t *testing.T) {
	h, companyID, solID := companyRoleFixture(t)
	mustCreateRole(t, solutionRolesPath(solID), "global_viewer", adminHeader())
	mustCreateRole(t, companyRolesPath(companyID, solID), "my_custom", h)

	// Custom role company lain di solution yang sama tidak boleh terlihat.
	otherCompany := mustCreateCompany(t, adminHeader())
	mustSubscribe(t, otherCompany, mustCreatePlan(t, solID, "other", nil))
	mustCreateRole(t, companyRolesPath(otherCompany, solID), "their_custom", adminHeader())

	w := testSrv.GET(companyRolesPath(companyID, solID), h)
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, []string{"global_viewer", "my_custom"}, roleCodes(t, w.Body.Bytes()), "global dulu, lalu custom milik company")
	assert.Equal(t, 401, testSrv.GET(companyRolesPath(companyID, solID)).Code)
}

func TestCompanyRole_ReplacePermissions(t *testing.T) {
	h, companyID, solID := companyRoleFixture(t)
	roleID := mustCreateRole(t, companyRolesPath(companyID, solID), "supervisor", h)
	globalRole := mustCreateRole(t, solutionRolesPath(solID), "global_role", adminHeader())

	body := map[string]any{"permission_codes": []string{"employee.read"}}
	assert.Equal(t, 404, testSrv.PUT(companyRolePermissionsPath(companyID, globalRole), body, h).Code, "role global lewat path company")
	assert.Equal(t, 404, testSrv.PUT(rolePermissionsPath(roleID), body, adminHeader()).Code, "custom role lewat path global")

	assert.Equal(t, 422, testSrv.PUT(companyRolePermissionsPath(companyID, roleID), map[string]any{"permission_codes": []string{"payroll.run"}}, h).Code)

	w := testSrv.PUT(companyRolePermissionsPath(companyID, roleID), body, h)
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, []string{"employee.read"}, permissionCodesOf(t, w.Body.Bytes()))
	assert.Equal(t, 1, countEvents(t, "role.permissions_changed", companyID))

	// Company lain tidak bisa melihat / mengubah custom role ini.
	strangerH, _, strangerCompany := mustCompanyOwner(t)
	assert.Equal(t, 404, testSrv.PUT(companyRolePermissionsPath(strangerCompany, roleID), body, strangerH).Code)
	assert.Equal(t, 404, testSrv.GET(rolePermissionsPath(roleID), strangerH).Code)
	assert.Equal(t, 200, testSrv.GET(rolePermissionsPath(roleID), h).Code)
}

// ── Detail / Update / Delete ─────────────────────────────────────────────────

func rolePath(id string) string { return "/api/v1/roles/" + id }
func companyRolePath(companyID, roleID string) string {
	return "/api/v1/companies/" + companyID + "/roles/" + roleID
}

type roleDetail struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Description     *string `json:"description"`
	Scope           string  `json:"scope"`
	PermissionCount int     `json:"permission_count"`
	MemberCount     int     `json:"member_count"`
}

func TestRole_GetGlobal_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(rolePath(uuid.NewString())).Code)
}

func TestRole_GetGlobal_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.GET(rolePath(uuid.NewString()), adminHeader()).Code)
	// Custom role company lain → 404.
	h, companyID, solID := companyRoleFixture(t)
	custom := mustCreateRole(t, companyRolesPath(companyID, solID), "secret_role", h)
	strangerH, _, _ := mustCompanyOwner(t)
	assert.Equal(t, 404, testSrv.GET(rolePath(custom), strangerH).Code)
}

func TestRole_GetGlobal_Success(t *testing.T) {
	f := newAuthzFixture(t)
	f.assign(t, f.hrAdmin, "", "")
	w := testSrv.GET(rolePath(f.hrAdmin), adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got roleDetail
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "global", got.Scope)
	assert.Equal(t, 2, got.PermissionCount)
	assert.Equal(t, 1, got.MemberCount)
}

func TestRole_UpdateGlobal_Forbidden_NonAdmin(t *testing.T) {
	roleID := mustCreateRole(t, solutionRolesPath(roleSolution(t)), "viewer", adminHeader())
	tok, _, _ := mustLoginNewUser(t)
	assert.Equal(t, 403, testSrv.PATCH(rolePath(roleID), map[string]any{"name": "X"}, testhelper.BearerHeader(tok.AccessToken)).Code)
	assert.Equal(t, 403, testSrv.DELETE(rolePath(roleID), testhelper.BearerHeader(tok.AccessToken)).Code)
}

func TestRole_UpdateGlobal_SystemImmutable(t *testing.T) {
	roleID := mustCreateRole(t, solutionRolesPath(roleSolution(t)), "owner", adminHeader())
	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE roles SET is_system = TRUE WHERE id = $1`, roleID)
	require.NoError(t, err)
	assert.Equal(t, 403, testSrv.PATCH(rolePath(roleID), map[string]any{"name": "X"}, adminHeader()).Code)
	assert.Equal(t, 403, testSrv.DELETE(rolePath(roleID), adminHeader()).Code)
}

func TestRole_UpdateGlobal_InvalidPayload(t *testing.T) {
	roleID := mustCreateRole(t, solutionRolesPath(roleSolution(t)), "viewer", adminHeader())
	assert.Equal(t, 422, testSrv.PATCH(rolePath(roleID), map[string]any{"name": " "}, adminHeader()).Code)
}

func TestRole_UpdateGlobal_NotFound_CustomRole(t *testing.T) {
	h, companyID, solID := companyRoleFixture(t)
	custom := mustCreateRole(t, companyRolesPath(companyID, solID), "custom_x", h)
	assert.Equal(t, 404, testSrv.PATCH(rolePath(custom), map[string]any{"name": "X"}, adminHeader()).Code)
}

func TestRole_UpdateDeleteGlobal_Success(t *testing.T) {
	f := newAuthzFixture(t)
	f.assign(t, f.hrAdmin, "", "")

	w := testSrv.PATCH(rolePath(f.hrAdmin), map[string]any{"name": "HR Admin Baru", "description": "Kelola karyawan"}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got roleDetail
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "HR Admin Baru", got.Name)
	require.NotNil(t, got.Description)
	assert.Equal(t, "Kelola karyawan", *got.Description)

	require.Equal(t, 200, testSrv.DELETE(rolePath(f.hrAdmin), adminHeader()).Code)
	assert.Equal(t, 404, testSrv.GET(rolePath(f.hrAdmin), adminHeader()).Code)
	var n int
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM user_role_assignments WHERE role_id = $1`, f.hrAdmin).Scan(&n))
	assert.Equal(t, 0, n, "assignment ikut dihapus")
	assert.False(t, f.serviceCheck(t, "employee.read", "", "").Allowed)
}

func TestCompanyRole_Get_And_Holders(t *testing.T) {
	f := newAuthzFixture(t)
	f.assign(t, f.supervisor, "branch", f.b1)

	w := testSrv.GET(companyRolePath(f.companyID, f.supervisor), f.ownerH)
	require.Equal(t, 200, w.Code, w.Body.String())
	var got roleDetail
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "company", got.Scope)
	assert.Equal(t, 1, got.MemberCount)

	// Role global juga bisa dilihat dari konteks company.
	assert.Equal(t, 200, testSrv.GET(companyRolePath(f.companyID, f.hrAdmin), f.ownerH).Code)

	w = testSrv.GET(companyRolePath(f.companyID, f.supervisor)+"/members", f.ownerH)
	require.Equal(t, 200, w.Code, w.Body.String())
	var holders []struct {
		UserID    string `json:"user_id"`
		ScopeType string `json:"scope_type"`
		ScopeName string `json:"scope_name"`
	}
	testhelper.DecodeBody(t, w, &holders)
	require.Len(t, holders, 1)
	assert.Equal(t, f.staffID, holders[0].UserID)
	assert.Equal(t, "Branch B1", holders[0].ScopeName)
}

func TestCompanyRole_Get_Unauthenticated_Forbidden_NotFound(t *testing.T) {
	f := newAuthzFixture(t)
	assert.Equal(t, 401, testSrv.GET(companyRolePath(f.companyID, f.supervisor)).Code)
	strangerH, _, _ := mustCompanyOwner(t)
	assert.Equal(t, 403, testSrv.GET(companyRolePath(f.companyID, f.supervisor), strangerH).Code)
	assert.Equal(t, 404, testSrv.GET(companyRolePath(f.companyID, uuid.NewString()), f.ownerH).Code)
	// Custom role company lain → 404.
	other, otherCompany, otherSol := companyRoleFixture(t)
	foreignRole := mustCreateRole(t, companyRolesPath(otherCompany, otherSol), "foreign", other)
	assert.Equal(t, 404, testSrv.GET(companyRolePath(f.companyID, foreignRole), f.ownerH).Code)
}

func TestCompanyRole_UpdateDelete(t *testing.T) {
	f := newAuthzFixture(t)
	f.assign(t, f.supervisor, "branch", f.b1)

	assert.Equal(t, 401, testSrv.PATCH(companyRolePath(f.companyID, f.supervisor), map[string]any{"name": "X"}).Code)
	assert.Equal(t, 404, testSrv.PATCH(companyRolePath(f.companyID, f.hrAdmin), map[string]any{"name": "X"}, f.ownerH).Code, "role global")
	assert.Equal(t, 422, testSrv.PATCH(companyRolePath(f.companyID, f.supervisor), map[string]any{"name": ""}, f.ownerH).Code)

	w := testSrv.PATCH(companyRolePath(f.companyID, f.supervisor), map[string]any{"name": "Supervisor Cabang"}, f.ownerH)
	require.Equal(t, 200, w.Code, w.Body.String())
	var got roleDetail
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "Supervisor Cabang", got.Name)
	assert.Equal(t, 1, countEvents(t, "role.updated", f.companyID))

	assert.Equal(t, 404, testSrv.DELETE(companyRolePath(f.companyID, f.hrAdmin), f.ownerH).Code, "role global")
	require.Equal(t, 200, testSrv.DELETE(companyRolePath(f.companyID, f.supervisor), f.ownerH).Code)
	assert.Equal(t, 404, testSrv.GET(companyRolePath(f.companyID, f.supervisor), f.ownerH).Code)
	assert.False(t, f.serviceCheck(t, "attendance.edit", "branch", f.b1).Allowed)
	assert.Equal(t, 1, countEvents(t, "role.deleted", f.companyID))
}

func TestCompanyRole_List_KindAndCounts(t *testing.T) {
	f := newAuthzFixture(t)
	f.assign(t, f.supervisor, "branch", f.b1)

	var resp struct {
		Data []struct {
			Code            string `json:"code"`
			PermissionCount int    `json:"permission_count"`
			MemberCount     int    `json:"member_count"`
		} `json:"data"`
	}
	w := testSrv.GET(companyRolesPath(f.companyID, f.solutionID)+"?kind=custom", f.ownerH)
	require.Equal(t, 200, w.Code, w.Body.String())
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "supervisor", resp.Data[0].Code)
	assert.Equal(t, 1, resp.Data[0].PermissionCount)
	assert.Equal(t, 1, resp.Data[0].MemberCount)

	w = testSrv.GET(companyRolesPath(f.companyID, f.solutionID)+"?kind=global", f.ownerH)
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "hr_admin", resp.Data[0].Code)

	assert.Equal(t, 422, testSrv.GET(companyRolesPath(f.companyID, f.solutionID)+"?kind=system", f.ownerH).Code)
}
