package api_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/testhelper"
)

func assignPath(companyID, userID string) string {
	return "/api/v1/companies/" + companyID + "/users/" + userID + "/role-assignments"
}

// authzFixture: company (owner ber-token asli) subscribe solution berkatalog permission,
// role global hr_admin (employee.write, employee.read @company), custom role supervisor
// (attendance.edit @branch), branch B1, dan member kedua (staff).
type authzFixture struct {
	ownerH                              map[string]string
	companyID, solutionID, solutionCode string
	hrAdmin, supervisor                 string
	b1                                  string
	staffID, staffEmail                 string
	ownerID                             string
	svc                                 map[string]string
}

func newAuthzFixture(t *testing.T) authzFixture {
	t.Helper()
	var f authzFixture
	f.ownerH, f.ownerID, f.companyID = mustCompanyOwner(t)
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT s.solution_id, sol.code FROM subscriptions s JOIN solutions sol ON sol.id = s.solution_id WHERE s.company_id = $1`,
		f.companyID).Scan(&f.solutionID, &f.solutionCode))
	mustSync(t, f.solutionID, adminHeader(), catalog("v1",
		[2]string{"employee.read", "R"}, [2]string{"employee.write", "W"}, [2]string{"attendance.edit", "A"}))

	f.hrAdmin = mustCreateRole(t, solutionRolesPath(f.solutionID), "hr_admin", adminHeader())
	require.Equal(t, 200, testSrv.PUT(rolePermissionsPath(f.hrAdmin), permsBody([]string{"employee.write", "employee.read"}), adminHeader()).Code)
	f.supervisor = mustCreateRole(t, companyRolesPath(f.companyID, f.solutionID), "supervisor", f.ownerH)
	require.Equal(t, 200, testSrv.PUT(companyRolePermissionsPath(f.companyID, f.supervisor), scopedPermsBody("branch", "attendance.edit"), f.ownerH).Code)

	f.b1 = mustCreateBranch(t, f.ownerH, f.companyID, "B1")

	f.staffEmail = testhelper.UniqueEmail("staff")
	f.staffID = mustCreateUser(t, f.staffEmail)
	require.Equal(t, 201, testSrv.POST(membersPath(f.companyID), map[string]any{"user_id": f.staffID}, f.ownerH).Code)

	f.svc = testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "authz:check", "roles:write"), ""))
	return f
}

func (f authzFixture) assign(t *testing.T, roleID, scopeType, scopeID string) {
	t.Helper()
	w := testSrv.POST(assignPath(f.companyID, f.staffID), assignBody(roleID, scopeType, scopeID), f.ownerH)
	require.Equal(t, 201, w.Code, "assign body=%s", w.Body.String())
}

type checkResult struct {
	Permission string   `json:"permission"`
	ScopeType  string   `json:"scope_type"`
	Allowed    bool     `json:"allowed"`
	GrantedBy  []string `json:"granted_by"`
}

func (f authzFixture) check(t *testing.T, h map[string]string, body map[string]any) checkResult {
	t.Helper()
	w := testSrv.POST("/api/v1/authz/check", body, h)
	require.Equal(t, 200, w.Code, w.Body.String())
	var got checkResult
	testhelper.DecodeBody(t, w, &got)
	return got
}

func (f authzFixture) serviceCheck(t *testing.T, permission, scopeType, scopeID string) checkResult {
	t.Helper()
	body := map[string]any{"company_id": f.companyID, "user_id": f.staffID, "permission": permission}
	if scopeType != "" {
		body["scope_type"], body["scope_id"] = scopeType, scopeID
	}
	return f.check(t, f.svc, body)
}

// ── Role assignment ──────────────────────────────────────────────────────────

func TestRoleAssignment_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(assignPath(uuid.NewString(), uuid.NewString()), map[string]any{"role_id": uuid.NewString()}).Code)
}

func TestRoleAssignment_Forbidden(t *testing.T) {
	f := newAuthzFixture(t)
	strangerH, _, _ := mustCompanyOwner(t)
	assert.Equal(t, 403, testSrv.POST(assignPath(f.companyID, f.staffID), map[string]any{"role_id": f.hrAdmin}, strangerH).Code)

}

// GATE-SUBSCRIPTION (nonaktif sementara): role tetap bisa di-assign di company yang belum subscribe solution role.
func TestRoleAssignment_WithoutSubscription_Success(t *testing.T) {
	f := newAuthzFixture(t)
	unsubscribed := mustCreateCompany(t, adminHeader())
	userID := mustCreateUser(t, testhelper.UniqueEmail("u"))
	require.Equal(t, 201, testSrv.POST(membersPath(unsubscribed), map[string]any{"user_id": userID}, adminHeader()).Code)
	// assert.Equal(t, 403, testSrv.POST(assignPath(unsubscribed, userID), map[string]any{"role_id": f.hrAdmin}, adminHeader()).Code)
	assert.Equal(t, 201, testSrv.POST(assignPath(unsubscribed, userID), map[string]any{"role_id": f.hrAdmin}, adminHeader()).Code)
}

func TestRoleAssignment_Invalid(t *testing.T) {
	f := newAuthzFixture(t)
	otherCompany := mustCreateCompany(t, adminHeader())
	otherBranch := mustCreateBranch(t, adminHeader(), otherCompany, "X1")

	tests := map[string]struct {
		userID string
		body   map[string]any
	}{
		"bukan member":                   {uuid.NewString(), map[string]any{"role_id": f.hrAdmin}},
		"role tidak ada":                 {f.staffID, map[string]any{"role_id": uuid.NewString()}},
		"nilai scope untuk role company": {f.staffID, assignBody(f.hrAdmin, "branch", f.b1)},
		"company sebagai kunci scope":    {f.staffID, assignBody(f.hrAdmin, "company_x", otherCompany)},
		"branch company lain":            {f.staffID, assignBody(f.supervisor, "branch", otherBranch)},
		"branch bukan uuid":              {f.staffID, assignBody(f.supervisor, "branch", "not-a-uuid")},
		"nilai branch tidak dikirim":     {f.staffID, map[string]any{"role_id": f.supervisor}},
		"kunci scope tak dipakai role":   {f.staffID, map[string]any{"role_id": f.supervisor, "scope_values": map[string]string{"branch": f.b1, "region": "x"}}},
		"label untuk scope tak dipakai":  {f.staffID, map[string]any{"role_id": f.hrAdmin, "scope_labels": map[string]string{"self": "x"}}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			w := testSrv.POST(assignPath(f.companyID, tt.userID), tt.body, f.ownerH)
			assert.Equal(t, 422, w.Code, w.Body.String())
		})
	}
}

func TestRoleAssignment_CustomRoleOfOtherCompany(t *testing.T) {
	f := newAuthzFixture(t)
	otherCompany := mustCreateCompany(t, adminHeader())
	mustSubscribe(t, otherCompany, mustCreatePlan(t, f.solutionID, "other", nil))
	foreignCustom := mustCreateRole(t, companyRolesPath(otherCompany, f.solutionID), "foreign", adminHeader())

	w := testSrv.POST(assignPath(f.companyID, f.staffID), map[string]any{"role_id": foreignCustom}, f.ownerH)
	assert.Equal(t, 422, w.Code, w.Body.String())
}

func TestRoleAssignment_Success_AndDuplicate(t *testing.T) {
	f := newAuthzFixture(t)
	w := testSrv.POST(assignPath(f.companyID, f.staffID), map[string]any{"role_id": f.hrAdmin}, f.ownerH)
	require.Equal(t, 201, w.Code, w.Body.String())
	var got struct {
		ScopeValues map[string]string `json:"scope_values"`
		RoleCode    string            `json:"role_code"`
	}
	testhelper.DecodeBody(t, w, &got)
	assert.Empty(t, got.ScopeValues, "role ber-scope company tanpa nilai scope")
	assert.Equal(t, "hr_admin", got.RoleCode)
	// +1 dari company_admin yang otomatis diberikan ke pembuat company.
	assert.Equal(t, 2, countEvents(t, "role_assignment.created", f.companyID))

	assert.Equal(t, 409, testSrv.POST(assignPath(f.companyID, f.staffID), map[string]any{"role_id": f.hrAdmin}, f.ownerH).Code)

	// Role ber-scope branch: nilai berbeda = assignment lain; nilai sama = 409; lewat token service juga.
	b2 := mustCreateBranch(t, f.ownerH, f.companyID, "B2")
	f.assign(t, f.supervisor, "branch", f.b1)
	assert.Equal(t, 409, testSrv.POST(assignPath(f.companyID, f.staffID), assignBody(f.supervisor, "branch", f.b1), f.ownerH).Code)
	w = testSrv.POST(assignPath(f.companyID, f.staffID), assignBody(f.supervisor, "branch", b2), f.svc)
	assert.Equal(t, 201, w.Code, w.Body.String())
}

// ── Authz check ──────────────────────────────────────────────────────────────

func TestAuthzCheck_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST("/api/v1/authz/check", map[string]any{"permission": "x"}).Code)
}

func TestAuthzCheck_Invalid(t *testing.T) {
	f := newAuthzFixture(t)
	assert.Equal(t, 422, testSrv.POST("/api/v1/authz/check", map[string]any{"company_id": f.companyID}, f.svc).Code, "tanpa permission")
	assert.Equal(t, 422, testSrv.POST("/api/v1/authz/check", map[string]any{"permission": "employee.read"}, f.svc).Code, "service tanpa company & user")
	assert.Equal(t, 422, testSrv.POST("/api/v1/authz/check", map[string]any{
		"company_id": f.companyID, "user_id": f.staffID, "permission": "employee.read", "scope_type": "company", "scope_id": uuid.NewString(),
	}, f.svc).Code, "scope tidak dikenal")
	noScope := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "roles:write"), ""))
	assert.Equal(t, 403, testSrv.POST("/api/v1/authz/check", map[string]any{"company_id": f.companyID, "user_id": f.staffID, "permission": "x"}, noScope).Code)
}

func TestAuthzCheck_ScopeHierarchy(t *testing.T) {
	f := newAuthzFixture(t)
	otherCompany := mustCreateCompany(t, adminHeader())

	assert.False(t, f.serviceCheck(t, "employee.write", "", "").Allowed, "belum ada assignment")

	f.assign(t, f.supervisor, "branch", f.b1)
	f.assign(t, f.hrAdmin, "", "")

	got := f.serviceCheck(t, "employee.write", "company", f.companyID)
	assert.True(t, got.Allowed)
	assert.Equal(t, []string{"hr_admin"}, got.GrantedBy)
	assert.True(t, f.serviceCheck(t, "employee.write", "branch", f.b1).Allowed, "company berlaku untuk branch di bawahnya")
	w := testSrv.POST("/api/v1/authz/check", map[string]any{
		"company_id": f.companyID, "user_id": f.staffID, "permission": "employee.write",
		"scope_type": "company", "scope_id": otherCompany,
	}, f.svc)
	assert.Equal(t, 422, w.Code, "scope company di luar company target ditolak")

	assert.True(t, f.serviceCheck(t, "attendance.edit", "branch", f.b1).Allowed)
	assert.False(t, f.serviceCheck(t, "attendance.edit", "company", f.companyID).Allowed, "branch tidak naik ke company")
	assert.False(t, f.serviceCheck(t, "payroll.run", "branch", f.b1).Allowed, "permission tidak dikenal")
}

func TestAuthzCheck_DeniedWhenMemberInactiveOrSubscriptionCanceled(t *testing.T) {
	f := newAuthzFixture(t)
	f.assign(t, f.hrAdmin, "", "")
	require.True(t, f.serviceCheck(t, "employee.read", "", "").Allowed)

	_, err := testSrv.DB.ExecContext(context.Background(),
		`UPDATE company_members SET status = 'inactive' WHERE company_id = $1 AND user_id = $2`, f.companyID, f.staffID)
	require.NoError(t, err)
	assert.False(t, f.serviceCheck(t, "employee.read", "", "").Allowed, "member inactive")

	_, err = testSrv.DB.ExecContext(context.Background(),
		`UPDATE company_members SET status = 'active' WHERE company_id = $1 AND user_id = $2`, f.companyID, f.staffID)
	require.NoError(t, err)
	_, err = testSrv.DB.ExecContext(context.Background(), `UPDATE subscriptions SET status = 'canceled' WHERE company_id = $1`, f.companyID)
	require.NoError(t, err)
	// GATE-SUBSCRIPTION (nonaktif sementara): grant tidak hilang walau subscription dibatalkan.
	// assert.False(t, f.serviceCheck(t, "employee.read", "", "").Allowed, "subscription dibatalkan")
	assert.True(t, f.serviceCheck(t, "employee.read", "", "").Allowed, "subscription dibatalkan")
}

func TestAuthzCheck_UserToken_ChecksSelf(t *testing.T) {
	f := newAuthzFixture(t)
	f.assign(t, f.hrAdmin, "", "")

	tok := mustLogin(t, f.staffEmail, f.solutionCode) // satu company → cid otomatis
	require.NotNil(t, tok.CompanyID)
	h := testhelper.BearerHeader(tok.AccessToken)

	got := f.check(t, h, map[string]any{"permission": "employee.write", "scope_type": "company", "scope_id": f.companyID})
	assert.True(t, got.Allowed)
	assert.True(t, f.check(t, h, map[string]any{"permission": "employee.write"}).Allowed)

	// User tidak boleh mengecek user lain.
	ownerID := uuid.NewString()
	assert.Equal(t, 403, testSrv.POST("/api/v1/authz/check", map[string]any{"permission": "employee.write", "user_id": ownerID}, h).Code)

	// Tanpa company aktif → 403.
	noCid := testhelper.BearerHeader(testSrv.Auth.Issue(principalUser(f.staffID, "")))
	assert.Equal(t, 403, testSrv.POST("/api/v1/authz/check", map[string]any{"permission": "employee.write"}, noCid).Code)
}

func TestAuthzCheckBatch(t *testing.T) {
	f := newAuthzFixture(t)
	f.assign(t, f.hrAdmin, "", "")

	w := testSrv.POST("/api/v1/authz/check-batch", map[string]any{
		"company_id": f.companyID, "user_id": f.staffID,
		"checks": []any{
			map[string]any{"permission": "employee.read", "scope_type": "company", "scope_id": f.companyID},
			map[string]any{"permission": "attendance.edit", "scope_type": "company", "scope_id": f.companyID},
			map[string]any{"permission": "attendance.edit", "scope_type": "branch", "scope_id": f.b1},
		},
	}, f.svc)
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp struct {
		Results []checkResult `json:"results"`
	}
	testhelper.DecodeBody(t, w, &resp)
	require.Len(t, resp.Results, 3)
	assert.Equal(t, []bool{true, false, false}, []bool{resp.Results[0].Allowed, resp.Results[1].Allowed, resp.Results[2].Allowed})

	tooMany := make([]any, 101)
	for i := range tooMany {
		tooMany[i] = map[string]any{"permission": "employee.read"}
	}
	assert.Equal(t, 422, testSrv.POST("/api/v1/authz/check-batch", map[string]any{
		"company_id": f.companyID, "user_id": f.staffID, "checks": tooMany,
	}, f.svc).Code)
	assert.Equal(t, 422, testSrv.POST("/api/v1/authz/check-batch", map[string]any{
		"company_id": f.companyID, "user_id": f.staffID, "checks": []any{},
	}, f.svc).Code)
}

func TestAuthzCheck_DeletedRoleNoLongerGrants(t *testing.T) {
	f := newAuthzFixture(t)
	f.assign(t, f.hrAdmin, "", "")
	require.True(t, f.serviceCheck(t, "employee.read", "", "").Allowed)

	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE roles SET deleted_at = NOW() WHERE id = $1`, f.hrAdmin)
	require.NoError(t, err)
	assert.False(t, f.serviceCheck(t, "employee.read", "", "").Allowed)
}

// ── List / Revoke assignment ─────────────────────────────────────────────────

type assignmentData struct {
	ID           string            `json:"id"`
	RoleCode     string            `json:"role_code"`
	RoleIsGlobal bool              `json:"role_is_global"`
	SolutionID   string            `json:"solution_id"`
	ScopeValues  map[string]string `json:"scope_values"`
	Scopes       []struct {
		Type     string `json:"type"`
		TypeName string `json:"type_name"`
		Value    string `json:"value"`
		Name     string `json:"name"`
	} `json:"scopes"`
	RoleScopeTypes []string `json:"role_scope_types"`
}

func listAssignments(t *testing.T, path string, h map[string]string) []assignmentData {
	t.Helper()
	w := testSrv.GET(path, h)
	require.Equal(t, 200, w.Code, w.Body.String())
	var got []assignmentData
	testhelper.DecodeBody(t, w, &got)
	return got
}

func TestRoleAssignment_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(assignPath(uuid.NewString(), uuid.NewString())).Code)
}

func TestRoleAssignment_List_Forbidden(t *testing.T) {
	f := newAuthzFixture(t)
	strangerH, _, _ := mustCompanyOwner(t)
	assert.Equal(t, 403, testSrv.GET(assignPath(f.companyID, f.staffID), strangerH).Code)
}

func TestRoleAssignment_List_InvalidQuery(t *testing.T) {
	f := newAuthzFixture(t)
	assert.Equal(t, 422, testSrv.GET(assignPath(f.companyID, f.staffID)+"?solution_id=x", f.ownerH).Code)
}

func TestRoleAssignment_List_Success(t *testing.T) {
	f := newAuthzFixture(t)
	f.assign(t, f.hrAdmin, "", "")
	f.assign(t, f.supervisor, "branch", f.b1)

	got := listAssignments(t, assignPath(f.companyID, f.staffID)+"?solution_id="+f.solutionID, f.ownerH)
	require.Len(t, got, 2)
	byRole := map[string]assignmentData{}
	for _, a := range got {
		byRole[a.RoleCode] = a
	}
	assert.True(t, byRole["hr_admin"].RoleIsGlobal)
	assert.Empty(t, byRole["hr_admin"].Scopes)
	assert.False(t, byRole["supervisor"].RoleIsGlobal)
	assert.Equal(t, map[string]string{"branch": f.b1}, byRole["supervisor"].ScopeValues)
	require.Len(t, byRole["supervisor"].Scopes, 1)
	assert.Equal(t, "Branch B1", byRole["supervisor"].Scopes[0].Name)

	// Filter solution lain → kosong; token service solution lain hanya melihat solution-nya.
	other := mustCreateSolution(t, testhelper.UniqueCode("oth"))
	assert.Empty(t, listAssignments(t, assignPath(f.companyID, f.staffID)+"?solution_id="+other, f.ownerH))
	foreign := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, other, "roles:write"), ""))
	assert.Empty(t, listAssignments(t, assignPath(f.companyID, f.staffID), foreign))
	assert.Len(t, listAssignments(t, assignPath(f.companyID, f.staffID), f.svc), 2)
}

func revokePath(id string) string {
	return "/api/v1/role-assignments/" + id
}

func TestRoleAssignment_Revoke_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.DELETE(revokePath(uuid.NewString())).Code)
}

func TestRoleAssignment_Revoke_NotFound(t *testing.T) {
	f := newAuthzFixture(t)
	assert.Equal(t, 404, testSrv.DELETE(revokePath(uuid.NewString()), f.ownerH).Code)
	assert.Equal(t, 422, testSrv.DELETE(revokePath("bukan-uuid"), f.ownerH).Code)

	// Assignment milik company lain / solution lain → 404 (keberadaannya tidak dibocorkan).
	f.assign(t, f.hrAdmin, "", "")
	id := listAssignments(t, assignPath(f.companyID, f.staffID), f.ownerH)[0].ID
	strangerH, _, _ := mustCompanyOwner(t)
	assert.Equal(t, 404, testSrv.DELETE(revokePath(id), strangerH).Code)
	other := mustCreateSolution(t, testhelper.UniqueCode("oth"))
	foreign := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, other, "roles:write"), ""))
	assert.Equal(t, 404, testSrv.DELETE(revokePath(id), foreign).Code)
}

func TestRoleAssignment_Revoke_Success(t *testing.T) {
	f := newAuthzFixture(t)
	f.assign(t, f.hrAdmin, "", "")
	require.True(t, f.serviceCheck(t, "employee.read", "", "").Allowed)
	id := listAssignments(t, assignPath(f.companyID, f.staffID), f.ownerH)[0].ID

	w := testSrv.DELETE(revokePath(id), f.ownerH)
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Empty(t, listAssignments(t, assignPath(f.companyID, f.staffID), f.ownerH))
	assert.False(t, f.serviceCheck(t, "employee.read", "", "").Allowed)
	assert.Equal(t, 1, countEvents(t, "role_assignment.deleted", f.companyID))

	assert.Equal(t, 404, testSrv.DELETE(revokePath(id), f.ownerH).Code, "sudah dihapus")
}

// ── Effective permissions ────────────────────────────────────────────────────

func effectivePath(companyID, userID string) string {
	return "/api/v1/companies/" + companyID + "/users/" + userID + "/effective-permissions"
}

type effectivePermission struct {
	Permission string `json:"permission"`
	Grants     []struct {
		RoleCode  string `json:"role_code"`
		ScopeType string `json:"scope_type"`
		ScopeID   string `json:"scope_id"`
	} `json:"grants"`
}

func TestEffectivePermissions_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(effectivePath(uuid.NewString(), uuid.NewString())).Code)
}

func TestEffectivePermissions_Forbidden(t *testing.T) {
	f := newAuthzFixture(t)
	strangerH, _, _ := mustCompanyOwner(t)
	assert.Equal(t, 403, testSrv.GET(effectivePath(f.companyID, f.staffID)+"?solution_id="+f.solutionID, strangerH).Code)
	noScope := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "roles:write"), ""))
	assert.Equal(t, 403, testSrv.GET(effectivePath(f.companyID, f.staffID), noScope).Code, "tanpa authz:check")
}

func TestEffectivePermissions_InvalidQuery(t *testing.T) {
	f := newAuthzFixture(t)
	assert.Equal(t, 422, testSrv.GET(effectivePath(f.companyID, f.staffID), f.ownerH).Code, "token user tanpa solution_id")
	assert.Equal(t, 422, testSrv.GET(effectivePath(f.companyID, f.staffID)+"?solution_id=x", f.ownerH).Code)
}

func TestEffectivePermissions_NotFound(t *testing.T) {
	f := newAuthzFixture(t)
	assert.Equal(t, 404, testSrv.GET(effectivePath(f.companyID, f.staffID)+"?solution_id="+uuid.NewString(), f.ownerH).Code)
}

func TestEffectivePermissions_Success(t *testing.T) {
	f := newAuthzFixture(t)
	f.assign(t, f.hrAdmin, "", "")
	f.assign(t, f.supervisor, "branch", f.b1)

	w := testSrv.GET(effectivePath(f.companyID, f.staffID)+"?solution_id="+f.solutionID, f.ownerH)
	require.Equal(t, 200, w.Code, w.Body.String())
	var got []effectivePermission
	testhelper.DecodeBody(t, w, &got)
	codes := make([]string, 0, len(got))
	for _, p := range got {
		codes = append(codes, p.Permission)
	}
	assert.Equal(t, []string{"attendance.edit", "employee.read", "employee.write"}, codes)
	require.Len(t, got[0].Grants, 1)
	assert.Equal(t, "supervisor", got[0].Grants[0].RoleCode)
	assert.Equal(t, "branch", got[0].Grants[0].ScopeType)
	assert.Equal(t, f.b1, got[0].Grants[0].ScopeID)

	// Token service memakai solution client tanpa solution_id.
	w = testSrv.GET(effectivePath(f.companyID, f.staffID), f.svc)
	require.Equal(t, 200, w.Code, w.Body.String())
	testhelper.DecodeBody(t, w, &got)
	assert.Len(t, got, 3)
}

// ── Scope custom (scope di role_permissions, nilai di assignment) ───────────

// customScopeFixture memperluas authzFixture: katalog berisi scope type department (parent
// branch) & self; role manager = items.read@company + profile.view@self; role kadep =
// leave.approve@department; role kacab = leave.approve@branch; role mixed = leave.approve@department
// + attendance.edit@branch.
type customScopeFixture struct {
	authzFixture
	manager, kadep, kacab, mixed string
}

func newCustomScopeFixture(t *testing.T) customScopeFixture {
	t.Helper()
	f := customScopeFixture{authzFixture: newAuthzFixture(t)}
	mustSync(t, f.solutionID, adminHeader(), map[string]any{
		"version": "v-custom",
		"scope_types": []any{
			map[string]any{"code": "department", "name": "Departemen", "parent": "branch"},
			map[string]any{"code": "self", "name": "Diri sendiri"},
		},
		"permissions": []any{
			map[string]any{"code": "employee.read", "name": "R"},
			map[string]any{"code": "employee.write", "name": "W"},
			map[string]any{"code": "attendance.edit", "name": "A"},
			map[string]any{"code": "leave.approve", "name": "Setujui cuti"},
			map[string]any{"code": "items.read", "name": "Lihat item"},
			map[string]any{"code": "profile.view", "name": "Lihat profil"},
		},
	})
	mk := func(code string, perms ...map[string]any) string {
		id := mustCreateRole(t, solutionRolesPath(f.solutionID), code, adminHeader())
		w := testSrv.PUT(rolePermissionsPath(id), map[string]any{"permissions": perms}, adminHeader())
		require.Equal(t, 200, w.Code, "role %s: %s", code, w.Body.String())
		return id
	}
	p := func(code, scope string) map[string]any { return map[string]any{"code": code, "scope_type": scope} }
	f.manager = mk("manager", p("items.read", ""), p("profile.view", "self"))
	f.kadep = mk("kadep", p("leave.approve", "department"))
	f.kacab = mk("kacab", p("leave.approve", "branch"))
	f.mixed = mk("mixed", p("leave.approve", "department"), p("attendance.edit", "branch"))
	return f
}

func (f customScopeFixture) assignValues(t *testing.T, roleID string, values, labels map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]any{"role_id": roleID, "scope_values": values}
	if labels != nil {
		body["scope_labels"] = labels
	}
	return testSrv.POST(assignPath(f.companyID, f.staffID), body, f.ownerH)
}

func (f customScopeFixture) checkScope(t *testing.T, permission, scopeType, scopeID, parentID string) checkResult {
	t.Helper()
	body := map[string]any{"company_id": f.companyID, "user_id": f.staffID, "permission": permission,
		"scope_type": scopeType, "scope_id": scopeID}
	if parentID != "" {
		body["parent_id"] = parentID
	}
	return f.check(t, f.svc, body)
}

func TestRolePermissions_ScopeType(t *testing.T) {
	f := newCustomScopeFixture(t)
	w := testSrv.GET(rolePermissionsPath(f.manager), f.ownerH)
	require.Equal(t, 200, w.Code, w.Body.String())
	var got []struct {
		Code      string `json:"code"`
		ScopeType string `json:"scope_type"`
	}
	testhelper.DecodeBody(t, w, &got)
	scopes := map[string]string{}
	for _, p := range got {
		scopes[p.Code] = p.ScopeType
	}
	assert.Equal(t, map[string]string{"items.read": "company", "profile.view": "self"}, scopes)

	// scope_type tak dikenal / scope type deprecated → 422.
	assert.Equal(t, 422, testSrv.PUT(rolePermissionsPath(f.manager), scopedPermsBody("region", "items.read"), adminHeader()).Code)
	mustSync(t, f.solutionID, adminHeader(), map[string]any{
		"version": "v-noself", "scope_types": []any{map[string]any{"code": "department", "name": "Departemen", "parent": "branch"}},
		"permissions": []any{map[string]any{"code": "items.read", "name": "I"}, map[string]any{"code": "profile.view", "name": "P"},
			map[string]any{"code": "leave.approve", "name": "L"}, map[string]any{"code": "attendance.edit", "name": "A"}},
	})
	assert.Equal(t, 422, testSrv.PUT(rolePermissionsPath(f.kadep), scopedPermsBody("self", "leave.approve"), adminHeader()).Code,
		"scope type deprecated tidak bisa dipasang baru")
	assert.Equal(t, 200, testSrv.PUT(rolePermissionsPath(f.manager), map[string]any{"permissions": []any{
		map[string]any{"code": "items.read"}, map[string]any{"code": "profile.view", "scope_type": "self"},
	}}, adminHeader()).Code, "yang sudah terpasang dengan scope deprecated boleh tetap")
}

func TestRoleAssignment_ScopeValues_Success(t *testing.T) {
	f := newCustomScopeFixture(t)
	w := f.assignValues(t, f.mixed, map[string]string{"department": " 42 ", "branch": f.b1}, map[string]string{"department": "Keuangan"})
	require.Equal(t, 201, w.Code, w.Body.String())
	var got struct {
		ScopeValues map[string]string `json:"scope_values"`
		ScopeLabels map[string]string `json:"scope_labels"`
	}
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, map[string]string{"department": "42", "branch": f.b1}, got.ScopeValues)
	assert.Equal(t, map[string]string{"department": "Keuangan"}, got.ScopeLabels)

	list := listAssignments(t, assignPath(f.companyID, f.staffID)+"?solution_id="+f.solutionID, f.ownerH)
	require.Len(t, list, 1)
	require.Len(t, list[0].Scopes, 2)
	assert.Equal(t, "branch", list[0].Scopes[0].Type)
	assert.Equal(t, "Branch B1", list[0].Scopes[0].Name)
	assert.Equal(t, "Departemen", list[0].Scopes[1].TypeName)
	assert.Equal(t, "Keuangan", list[0].Scopes[1].Name)
	assert.ElementsMatch(t, []string{"branch", "department"}, list[0].RoleScopeTypes)

	// Nilai berbeda = assignment lain; nilai sama = 409.
	assert.Equal(t, 201, f.assignValues(t, f.mixed, map[string]string{"department": "43", "branch": f.b1}, nil).Code)
	assert.Equal(t, 409, f.assignValues(t, f.mixed, map[string]string{"department": "42", "branch": f.b1}, nil).Code)
}

func TestRoleAssignment_ScopeValues_Invalid(t *testing.T) {
	f := newCustomScopeFixture(t)
	otherBranch := mustCreateBranch(t, adminHeader(), mustCreateCompany(t, adminHeader()), "X1")
	tests := map[string]map[string]string{
		"tanpa nilai self":       nil,
		"nilai kosong":           {"self": "  "},
		"kunci tidak dipakai":    {"self": "u1", "department": "1"},
		"branch company lain":    {"branch": otherBranch, "department": "1"},
		"department tanpa nilai": {"branch": f.b1},
	}
	for name, values := range tests {
		t.Run(name, func(t *testing.T) {
			role := f.manager
			if name == "branch company lain" || name == "department tanpa nilai" {
				role = f.mixed
			}
			w := f.assignValues(t, role, values, nil)
			assert.Equal(t, 422, w.Code, w.Body.String())
		})
	}

	// Scope type deprecated tidak bisa dipakai assignment baru.
	mustSync(t, f.solutionID, adminHeader(), map[string]any{
		"version": "v-noself", "permissions": []any{map[string]any{"code": "items.read", "name": "I"},
			map[string]any{"code": "profile.view", "name": "P"}, map[string]any{"code": "leave.approve", "name": "L"},
			map[string]any{"code": "attendance.edit", "name": "A"}},
	})
	assert.Equal(t, 422, f.assignValues(t, f.manager, map[string]string{"self": "u1"}, nil).Code)
}

// Kasus user: role manager berisi permission tanpa scope (company) + satu permission scope self.
// Permission company tetap berlaku di seluruh company; hanya profile.view yang terbatas ke self.
func TestAuthzCheck_RoleWithMixedScopes(t *testing.T) {
	f := newCustomScopeFixture(t)
	require.Equal(t, 201, f.assignValues(t, f.manager, map[string]string{"self": "u-1"}, nil).Code)

	assert.True(t, f.checkScope(t, "items.read", "company", f.companyID, "").Allowed, "permission company berlaku di company")
	assert.True(t, f.checkScope(t, "items.read", "branch", f.b1, "").Allowed, "... dan di semua branch")
	assert.True(t, f.checkScope(t, "items.read", "self", "u-99", "").Allowed, "... dan di semua scope custom")
	assert.True(t, f.checkScope(t, "profile.view", "self", "u-1", "").Allowed)
	assert.False(t, f.checkScope(t, "profile.view", "self", "u-2", "").Allowed, "nilai self lain")
	assert.False(t, f.checkScope(t, "profile.view", "company", f.companyID, "").Allowed, "self tidak naik ke company")

	w := testSrv.GET(effectivePath(f.companyID, f.staffID), f.svc)
	require.Equal(t, 200, w.Code, w.Body.String())
	var eff []effectivePermission
	testhelper.DecodeBody(t, w, &eff)
	byPerm := map[string]effectivePermission{}
	for _, p := range eff {
		byPerm[p.Permission] = p
	}
	assert.Equal(t, "company", byPerm["items.read"].Grants[0].ScopeType)
	assert.Equal(t, "self", byPerm["profile.view"].Grants[0].ScopeType)
	assert.Equal(t, "u-1", byPerm["profile.view"].Grants[0].ScopeID)
}

func TestAuthzCheck_CustomScopeHierarchy(t *testing.T) {
	f := newCustomScopeFixture(t)
	b2 := mustCreateBranch(t, f.ownerH, f.companyID, "B2")
	require.Equal(t, 201, f.assignValues(t, f.kadep, map[string]string{"department": "D1"}, nil).Code)
	require.Equal(t, 201, f.assignValues(t, f.kacab, map[string]string{"branch": f.b1}, nil).Code)

	got := f.checkScope(t, "leave.approve", "department", "D1", "")
	assert.True(t, got.Allowed)
	assert.Equal(t, []string{"kadep"}, got.GrantedBy)
	assert.False(t, f.checkScope(t, "leave.approve", "department", "D2", "").Allowed, "department lain")
	assert.Equal(t, []string{"kacab"}, f.checkScope(t, "leave.approve", "department", "D2", f.b1).GrantedBy,
		"grant branch menutupi department ber-parent branch itu")
	assert.False(t, f.checkScope(t, "leave.approve", "department", "D2", b2).Allowed, "branch lain")
	assert.True(t, f.checkScope(t, "leave.approve", "branch", f.b1, "").Allowed)
	assert.False(t, f.checkScope(t, "leave.approve", "branch", b2, "").Allowed)
	assert.False(t, f.checkScope(t, "leave.approve", "company", f.companyID, "").Allowed)

	// Scope type tak dikenal / parent untuk scope ber-parent company → 422.
	w := testSrv.POST("/api/v1/authz/check", map[string]any{"company_id": f.companyID, "user_id": f.staffID,
		"permission": "leave.approve", "scope_type": "region", "scope_id": "1"}, f.svc)
	assert.Equal(t, 422, w.Code)
	w = testSrv.POST("/api/v1/authz/check", map[string]any{"company_id": f.companyID, "user_id": f.staffID,
		"permission": "profile.view", "scope_type": "self", "scope_id": "1", "parent_id": f.b1}, f.svc)
	assert.Equal(t, 422, w.Code)
}

func TestAuthzCheck_RoleChangedAfterAssignment(t *testing.T) {
	f := newCustomScopeFixture(t)
	require.Equal(t, 201, f.assignValues(t, f.kacab, map[string]string{"branch": f.b1}, nil).Code)
	// Role diubah: permission dipindah ke scope self → assignment lama tidak punya nilai self.
	require.Equal(t, 200, testSrv.PUT(rolePermissionsPath(f.kacab), scopedPermsBody("self", "leave.approve"), adminHeader()).Code)
	assert.False(t, f.checkScope(t, "leave.approve", "branch", f.b1, "").Allowed)
	assert.False(t, f.checkScope(t, "leave.approve", "self", f.b1, "").Allowed)
}

func TestRoleAssignment_BranchAdminScope(t *testing.T) {
	f := newCustomScopeFixture(t)
	consoleID := mustSolutionID(t, testConsoleSolution)
	// Staff memegang role.assign hanya di cabang B1.
	w := testSrv.POST("/api/v1/companies/"+f.companyID+"/solutions/"+consoleID+"/roles", map[string]any{"code": "assigner", "name": "A"}, f.ownerH)
	require.Equal(t, 201, w.Code, w.Body.String())
	var role struct {
		ID string `json:"id"`
	}
	testhelper.DecodeBody(t, w, &role)
	require.Equal(t, 200, testSrv.PUT(companyRolePermissionsPath(f.companyID, role.ID), scopedPermsBody("branch", "role.assign"), f.ownerH).Code)
	f.assign(t, role.ID, "branch", f.b1)
	staffTok := mustLogin(t, f.staffEmail, testConsoleSolution)
	staffH := testhelper.BearerHeader(staffTok.AccessToken)

	ownerID := f.ownerID
	b2 := mustCreateBranch(t, f.ownerH, f.companyID, "B2")
	post := func(body map[string]any) int {
		return testSrv.POST(assignPath(f.companyID, ownerID), body, staffH).Code
	}
	assert.Equal(t, 201, post(assignBody(f.kacab, "branch", f.b1)), "role ber-scope branch di cabangnya")
	assert.Equal(t, 403, post(assignBody(f.kacab, "branch", b2)), "cabang lain")
	assert.Equal(t, 403, post(map[string]any{"role_id": f.hrAdmin}), "role ber-scope company butuh level company")
	assert.Equal(t, 403, post(map[string]any{"role_id": f.kadep, "scope_values": map[string]string{"department": "D1"}}), "scope custom butuh level company")
}

func TestEffectivePermissions_IncludesNameAndGroup(t *testing.T) {
	f := newAuthzFixture(t)
	mustSync(t, f.solutionID, adminHeader(), map[string]any{
		"version": "v-groups",
		"groups":  []any{map[string]any{"code": "employee", "name": "Karyawan"}},
		"permissions": []any{
			map[string]any{"code": "employee.read", "name": "Lihat karyawan", "group": "employee"},
			map[string]any{"code": "employee.write", "name": "Ubah karyawan", "group": "employee"},
			map[string]any{"code": "attendance.edit", "name": "Ubah absensi"},
		},
	})
	f.assign(t, f.hrAdmin, "", "")

	w := testSrv.GET(effectivePath(f.companyID, f.staffID)+"?solution_id="+f.solutionID, f.ownerH)
	require.Equal(t, 200, w.Code, w.Body.String())
	var got []struct {
		Permission string `json:"permission"`
		Name       string `json:"name"`
		Group      *struct {
			Name string `json:"name"`
		} `json:"group"`
	}
	testhelper.DecodeBody(t, w, &got)
	require.Len(t, got, 2)
	assert.Equal(t, "Lihat karyawan", got[0].Name)
	require.NotNil(t, got[0].Group)
	assert.Equal(t, "Karyawan", got[0].Group.Name)
	assert.Equal(t, "Ubah karyawan", got[1].Name)
}
