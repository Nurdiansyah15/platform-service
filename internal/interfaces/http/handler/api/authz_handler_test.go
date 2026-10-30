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
