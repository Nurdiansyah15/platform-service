package api_test

import (
	"context"
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
// role global hr_admin (employee.write, employee.read), custom role supervisor
// (attendance.edit), branch B1, dan member kedua (staff).
type authzFixture struct {
	ownerH                              map[string]string
	companyID, solutionID, solutionCode string
	hrAdmin, supervisor                 string
	b1                                  string
	staffID, staffEmail                 string
	svc                                 map[string]string
}

func newAuthzFixture(t *testing.T) authzFixture {
	t.Helper()
	var f authzFixture
	f.ownerH, _, f.companyID = mustCompanyOwner(t)
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT s.solution_id, sol.code FROM subscriptions s JOIN solutions sol ON sol.id = s.solution_id WHERE s.company_id = $1`,
		f.companyID).Scan(&f.solutionID, &f.solutionCode))
	mustSync(t, f.solutionID, adminHeader(), catalog("v1",
		[2]string{"employee.read", "R"}, [2]string{"employee.write", "W"}, [2]string{"attendance.edit", "A"}))

	f.hrAdmin = mustCreateRole(t, solutionRolesPath(f.solutionID), "hr_admin", adminHeader())
	require.Equal(t, 200, testSrv.PUT(rolePermissionsPath(f.hrAdmin), map[string]any{"permission_codes": []string{"employee.write", "employee.read"}}, adminHeader()).Code)
	f.supervisor = mustCreateRole(t, companyRolesPath(f.companyID, f.solutionID), "supervisor", f.ownerH)
	require.Equal(t, 200, testSrv.PUT(companyRolePermissionsPath(f.companyID, f.supervisor), map[string]any{"permission_codes": []string{"attendance.edit"}}, f.ownerH).Code)

	f.b1 = mustCreateBranch(t, f.ownerH, f.companyID, "B1")

	f.staffEmail = testhelper.UniqueEmail("staff")
	f.staffID = mustCreateUser(t, f.staffEmail)
	require.Equal(t, 201, testSrv.POST(membersPath(f.companyID), map[string]any{"user_id": f.staffID}, f.ownerH).Code)

	f.svc = testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "authz:check", "roles:write"), ""))
	return f
}

func (f authzFixture) assign(t *testing.T, roleID, scopeType, scopeID string) {
	t.Helper()
	body := map[string]any{"role_id": roleID}
	if scopeType != "" {
		body["scope_type"], body["scope_id"] = scopeType, scopeID
	}
	w := testSrv.POST(assignPath(f.companyID, f.staffID), body, f.ownerH)
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
		"bukan member":           {uuid.NewString(), map[string]any{"role_id": f.hrAdmin}},
		"role tidak ada":         {f.staffID, map[string]any{"role_id": uuid.NewString()}},
		"company company lain":   {f.staffID, map[string]any{"role_id": f.hrAdmin, "scope_type": "company", "scope_id": otherCompany}},
		"branch company lain":    {f.staffID, map[string]any{"role_id": f.hrAdmin, "scope_type": "branch", "scope_id": otherBranch}},
		"branch tanpa scope_id":  {f.staffID, map[string]any{"role_id": f.hrAdmin, "scope_type": "branch"}},
		"scope_type tak dikenal": {f.staffID, map[string]any{"role_id": f.hrAdmin, "scope_type": "region", "scope_id": f.b1}},
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
		ScopeType string `json:"scope_type"`
		ScopeID   string `json:"scope_id"`
		RoleCode  string `json:"role_code"`
	}
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "company", got.ScopeType)
	assert.Equal(t, f.companyID, got.ScopeID, "scope company default = company itu sendiri")
	assert.Equal(t, "hr_admin", got.RoleCode)
	// +1 dari company_admin yang otomatis diberikan ke pembuat company.
	assert.Equal(t, 2, countEvents(t, "role_assignment.created", f.companyID))

	assert.Equal(t, 409, testSrv.POST(assignPath(f.companyID, f.staffID), map[string]any{"role_id": f.hrAdmin}, f.ownerH).Code)

	// Role sama di scope lain boleh; custom role company sendiri boleh; lewat token service juga.
	f.assign(t, f.hrAdmin, "branch", f.b1)
	w = testSrv.POST(assignPath(f.companyID, f.staffID), map[string]any{"role_id": f.supervisor, "scope_type": "branch", "scope_id": f.b1}, f.svc)
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
	ID           string `json:"id"`
	RoleCode     string `json:"role_code"`
	RoleIsGlobal bool   `json:"role_is_global"`
	SolutionID   string `json:"solution_id"`
	ScopeType    string `json:"scope_type"`
	ScopeID      string `json:"scope_id"`
	ScopeName    string `json:"scope_name"`
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
	assert.Equal(t, "company", byRole["hr_admin"].ScopeType)
	assert.False(t, byRole["supervisor"].RoleIsGlobal)
	assert.Equal(t, f.b1, byRole["supervisor"].ScopeID)
	assert.Equal(t, "Branch B1", byRole["supervisor"].ScopeName)

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

// ── Scope custom ─────────────────────────────────────────────────────────────

// customScopeFixture memperluas authzFixture: katalog solution berisi scope type department
// (parent branch) & project (parent company); role kadep (leave.approve) dan role payroll
// (payroll.run yang hanya berlaku di company).
type customScopeFixture struct {
	authzFixture
	kadep, payroll, viewer string
}

func newCustomScopeFixture(t *testing.T) customScopeFixture {
	t.Helper()
	f := customScopeFixture{authzFixture: newAuthzFixture(t)}
	mustSync(t, f.solutionID, adminHeader(), map[string]any{
		"version": "v-custom",
		"scope_types": []any{
			map[string]any{"code": "department", "name": "Departemen", "parent": "branch"},
			map[string]any{"code": "project", "name": "Proyek"},
		},
		"permissions": []any{
			map[string]any{"code": "employee.read", "name": "R"},
			map[string]any{"code": "employee.write", "name": "W"},
			map[string]any{"code": "attendance.edit", "name": "A"},
			map[string]any{"code": "leave.approve", "name": "Setujui cuti"},
			map[string]any{"code": "payroll.run", "name": "Payroll", "scopes": []string{"company"}},
		},
	})
	f.kadep = mustCreateRole(t, solutionRolesPath(f.solutionID), "kadep", adminHeader())
	require.Equal(t, 200, testSrv.PUT(rolePermissionsPath(f.kadep), map[string]any{"permission_codes": []string{"leave.approve"}}, adminHeader()).Code)
	f.payroll = mustCreateRole(t, solutionRolesPath(f.solutionID), "payroll", adminHeader())
	require.Equal(t, 200, testSrv.PUT(rolePermissionsPath(f.payroll), map[string]any{"permission_codes": []string{"payroll.run"}}, adminHeader()).Code)
	f.viewer = mustCreateRole(t, solutionRolesPath(f.solutionID), "kacab", adminHeader())
	require.Equal(t, 200, testSrv.PUT(rolePermissionsPath(f.viewer), map[string]any{"permission_codes": []string{"leave.approve"}}, adminHeader()).Code)
	return f
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

func TestRoleAssignment_CustomScope_Success(t *testing.T) {
	f := newCustomScopeFixture(t)
	w := testSrv.POST(assignPath(f.companyID, f.staffID), map[string]any{
		"role_id": f.kadep, "scope_type": "department", "scope_id": "42", "parent_id": f.b1, "scope_label": " Keuangan ",
	}, f.ownerH)
	require.Equal(t, 201, w.Code, w.Body.String())
	var got struct {
		ScopeType  string  `json:"scope_type"`
		ScopeID    string  `json:"scope_id"`
		ScopeLabel *string `json:"scope_label"`
		ParentID   *string `json:"parent_id"`
	}
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "department", got.ScopeType)
	assert.Equal(t, "42", got.ScopeID)
	require.NotNil(t, got.ScopeLabel)
	assert.Equal(t, "Keuangan", *got.ScopeLabel)
	require.NotNil(t, got.ParentID)
	assert.Equal(t, f.b1, *got.ParentID)

	w = testSrv.GET(assignPath(f.companyID, f.staffID)+"?solution_id="+f.solutionID, f.ownerH)
	require.Equal(t, 200, w.Code, w.Body.String())
	var list []struct {
		ScopeType     string `json:"scope_type"`
		ScopeTypeName string `json:"scope_type_name"`
		ScopeName     string `json:"scope_name"`
	}
	testhelper.DecodeBody(t, w, &list)
	require.Len(t, list, 1)
	assert.Equal(t, "Departemen", list[0].ScopeTypeName)
	assert.Equal(t, "Keuangan", list[0].ScopeName)

	// Label kosong → scope_name = scope_id; scope ber-parent company tanpa parent_id.
	w = testSrv.POST(assignPath(f.companyID, f.staffID), map[string]any{
		"role_id": f.kadep, "scope_type": "project", "scope_id": "PRJ-7",
	}, f.svc)
	assert.Equal(t, 201, w.Code, w.Body.String())
}

func TestRoleAssignment_CustomScope_Invalid(t *testing.T) {
	f := newCustomScopeFixture(t)
	otherBranch := mustCreateBranch(t, adminHeader(), mustCreateCompany(t, adminHeader()), "X1")
	tests := map[string]map[string]any{
		"scope type tidak dikenal":          {"role_id": f.kadep, "scope_type": "region", "scope_id": "1"},
		"scope custom tanpa scope_id":       {"role_id": f.kadep, "scope_type": "department"},
		"parent untuk scope parent company": {"role_id": f.kadep, "scope_type": "project", "scope_id": "1", "parent_id": f.b1},
		"parent branch company lain":        {"role_id": f.kadep, "scope_type": "department", "scope_id": "1", "parent_id": otherBranch},
		"branch id bukan uuid":              {"role_id": f.kadep, "scope_type": "branch", "scope_id": "not-a-uuid"},
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			w := testSrv.POST(assignPath(f.companyID, f.staffID), body, f.ownerH)
			assert.Equal(t, 422, w.Code, w.Body.String())
		})
	}

	// Scope type milik solution lain tidak bisa dipakai role solution ini.
	otherSol := mustCreateSolution(t, testhelper.UniqueCode("os"))
	mustSync(t, otherSol, adminHeader(), map[string]any{
		"version": "v1", "permissions": []any{},
		"scope_types": []any{map[string]any{"code": "warehouse", "name": "Gudang"}},
	})
	w := testSrv.POST(assignPath(f.companyID, f.staffID), map[string]any{"role_id": f.kadep, "scope_type": "warehouse", "scope_id": "W1"}, f.ownerH)
	assert.Equal(t, 422, w.Code, w.Body.String())
}

func TestRoleAssignment_CustomScope_DeprecatedScopeType(t *testing.T) {
	f := newCustomScopeFixture(t)
	f.assignBody(t, map[string]any{"role_id": f.kadep, "scope_type": "project", "scope_id": "P1"})
	mustSync(t, f.solutionID, adminHeader(), catalog("v-noscope", [2]string{"leave.approve", "Setujui cuti"}))

	w := testSrv.POST(assignPath(f.companyID, f.staffID), map[string]any{"role_id": f.kadep, "scope_type": "project", "scope_id": "P2"}, f.ownerH)
	assert.Equal(t, 422, w.Code, "assignment baru ke scope type deprecated ditolak")
	assert.True(t, f.checkScope(t, "leave.approve", "project", "P1", "").Allowed, "assignment lama tetap dievaluasi")
}

func (f customScopeFixture) assignBody(t *testing.T, body map[string]any) {
	t.Helper()
	w := testSrv.POST(assignPath(f.companyID, f.staffID), body, f.ownerH)
	require.Equal(t, 201, w.Code, "assign body=%s", w.Body.String())
}

func TestAuthzCheck_CustomScope(t *testing.T) {
	f := newCustomScopeFixture(t)
	b2 := mustCreateBranch(t, f.ownerH, f.companyID, "B2")
	f.assignBody(t, map[string]any{"role_id": f.kadep, "scope_type": "department", "scope_id": "D1"})
	f.assignBody(t, map[string]any{"role_id": f.viewer, "scope_type": "branch", "scope_id": f.b1})

	got := f.checkScope(t, "leave.approve", "department", "D1", "")
	assert.True(t, got.Allowed)
	assert.Equal(t, []string{"kadep"}, got.GrantedBy)
	assert.Equal(t, "department", got.ScopeType)

	assert.False(t, f.checkScope(t, "leave.approve", "department", "D2", "").Allowed, "id lain")
	assert.False(t, f.checkScope(t, "leave.approve", "project", "D1", "").Allowed, "type lain")
	assert.Equal(t, []string{"kacab"}, f.checkScope(t, "leave.approve", "department", "D2", f.b1).GrantedBy,
		"grant branch menutupi department ber-parent branch itu")
	assert.False(t, f.checkScope(t, "leave.approve", "department", "D2", b2).Allowed, "branch lain")
	assert.False(t, f.checkScope(t, "leave.approve", "company", f.companyID, "").Allowed, "custom tidak naik ke company")

	// Company grant menutupi semua scope custom.
	f.assign(t, f.payroll, "", "")
	f.assign(t, f.hrAdmin, "", "")
	assert.True(t, f.checkScope(t, "employee.read", "project", "P9", "").Allowed)

	// Permission yang dibatasi scope company: cek di scope lain ditolak.
	assert.True(t, f.checkScope(t, "payroll.run", "company", f.companyID, "").Allowed)
	assert.False(t, f.checkScope(t, "payroll.run", "branch", f.b1, "").Allowed)
	assert.False(t, f.checkScope(t, "payroll.run", "department", "D1", "").Allowed)

	// Scope type tak dikenal / parent untuk scope ber-parent company → 422.
	w := testSrv.POST("/api/v1/authz/check", map[string]any{"company_id": f.companyID, "user_id": f.staffID,
		"permission": "leave.approve", "scope_type": "region", "scope_id": "1"}, f.svc)
	assert.Equal(t, 422, w.Code)
	w = testSrv.POST("/api/v1/authz/check", map[string]any{"company_id": f.companyID, "user_id": f.staffID,
		"permission": "leave.approve", "scope_type": "project", "scope_id": "1", "parent_id": f.b1}, f.svc)
	assert.Equal(t, 422, w.Code)
}

func TestAuthzCheck_PermissionScopeRestriction_GrantIgnored(t *testing.T) {
	f := newCustomScopeFixture(t)
	// payroll.run hanya berlaku di company → assignment branch tidak memberi apa pun.
	f.assignBody(t, map[string]any{"role_id": f.payroll, "scope_type": "branch", "scope_id": f.b1})
	assert.False(t, f.checkScope(t, "payroll.run", "branch", f.b1, "").Allowed)
	assert.False(t, f.checkScope(t, "payroll.run", "company", f.companyID, "").Allowed)
}

func TestEffectivePermissions_IncludesNameAndGroup(t *testing.T) {
	f := newAuthzFixture(t)
	mustSync(t, f.solutionID, adminHeader(), map[string]any{
		"version": "v-groups",
		"groups":  []any{map[string]any{"code": "employee", "name": "Karyawan"}},
		"permissions": []any{
			map[string]any{"code": "employee.read", "name": "Lihat karyawan", "group": "employee"},
			map[string]any{"code": "employee.write", "name": "Ubah karyawan", "group": "employee", "scopes": []string{"company"}},
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
		ScopeTypes []string `json:"scope_types"`
	}
	testhelper.DecodeBody(t, w, &got)
	require.Len(t, got, 2)
	assert.Equal(t, "Lihat karyawan", got[0].Name)
	require.NotNil(t, got[0].Group)
	assert.Equal(t, "Karyawan", got[0].Group.Name)
	assert.Equal(t, []string{"company"}, got[1].ScopeTypes)
}
