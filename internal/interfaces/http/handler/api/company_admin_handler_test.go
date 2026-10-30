package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/testhelper"
)

// adminFixture — company dengan owner (otomatis company_admin) dan staff tanpa role apa pun,
// keduanya memegang token ASLI ber-cid.
type adminFixture struct {
	ownerH, staffH     map[string]string
	ownerID, staffID   string
	companyID          string
	consoleSolutionID  string
	companyAdminRoleID string
}

func newAdminFixture(t *testing.T) adminFixture {
	t.Helper()
	ownerH, ownerID, companyID := mustCompanyOwner(t)
	email := testhelper.UniqueEmail("staff")
	staffID := mustCreateCompanyUser(t, ownerH, companyID, email)
	staff := mustLogin(t, email, testConsoleSolution)
	staffTok := mustSwitchCompany(t, staff.AccessToken, companyID)

	f := adminFixture{
		ownerH: ownerH, staffH: testhelper.BearerHeader(staffTok.AccessToken),
		ownerID: ownerID, staffID: staffID, companyID: companyID,
		consoleSolutionID: mustSolutionID(t, testConsoleSolution),
	}
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT id FROM roles WHERE solution_id = $1 AND code = 'company_admin' AND company_id IS NULL`,
		f.consoleSolutionID).Scan(&f.companyAdminRoleID))
	return f
}

// mustCreateCompanyUser membuat akun user baru sebagai member company lewat API.
func mustCreateCompanyUser(t *testing.T, headers map[string]string, companyID, email string) string {
	t.Helper()
	w := testSrv.POST("/api/v1/companies/"+companyID+"/users", map[string]any{
		"name": "User " + email, "email": email, "password": testPassword,
	}, headers)
	require.Equal(t, 201, w.Code, "mustCreateCompanyUser body=%s", w.Body.String())
	var got struct {
		UserID string `json:"user_id"`
	}
	testhelper.DecodeBody(t, w, &got)
	return got.UserID
}

// mustConsoleRole membuat custom role solution console milik company berisi permission platform.
func (f adminFixture) mustConsoleRole(t *testing.T, code string, perms ...string) string {
	t.Helper()
	w := testSrv.POST("/api/v1/companies/"+f.companyID+"/solutions/"+f.consoleSolutionID+"/roles",
		map[string]any{"code": code, "name": code}, f.ownerH)
	require.Equal(t, 201, w.Code, "create console role: %s", w.Body.String())
	var role struct {
		ID string `json:"id"`
	}
	testhelper.DecodeBody(t, w, &role)
	w = testSrv.PUT("/api/v1/companies/"+f.companyID+"/roles/"+role.ID+"/permissions",
		map[string]any{"permission_codes": perms}, f.ownerH)
	require.Equal(t, 200, w.Code, "set console role permissions: %s", w.Body.String())
	return role.ID
}

func (f adminFixture) assignStaff(t *testing.T, roleID, scopeType, scopeID string) {
	t.Helper()
	w := testSrv.POST(assignPath(f.companyID, f.staffID),
		map[string]any{"role_id": roleID, "scope_type": scopeType, "scope_id": scopeID}, f.ownerH)
	require.Equal(t, 201, w.Code, "assign staff: %s", w.Body.String())
}

// ── Create user ──────────────────────────────────────────────────────────────

func companyUsersPath(companyID string) string { return "/api/v1/companies/" + companyID + "/users" }

func TestCompany_CreateUser_Unauthenticated(t *testing.T) {
	_, _, companyID := mustCompanyOwner(t)
	assert.Equal(t, 401, testSrv.POST(companyUsersPath(companyID), map[string]any{}).Code)
}

func TestCompany_CreateUser_InvalidPayload(t *testing.T) {
	ownerH, _, companyID := mustCompanyOwner(t)
	w := testSrv.POST(companyUsersPath(companyID), map[string]any{"email": "x@y.co"}, ownerH)
	assert.Equal(t, 422, w.Code, w.Body.String())
	w = testSrv.POST(companyUsersPath(companyID), map[string]any{
		"name": "A", "email": "bukan-email", "password": testPassword,
	}, ownerH)
	assert.Equal(t, 422, w.Code, w.Body.String())
}

func TestCompany_CreateUser_Forbidden_NoAdminPermission(t *testing.T) {
	f := newAdminFixture(t)
	w := testSrv.POST(companyUsersPath(f.companyID), map[string]any{
		"name": "Baru", "email": testhelper.UniqueEmail("baru"), "password": testPassword,
	}, f.staffH)
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestCompany_CreateUser_Duplicate(t *testing.T) {
	ownerH, _, companyID := mustCompanyOwner(t)
	email := testhelper.UniqueEmail("dup")
	mustCreateUser(t, email)
	w := testSrv.POST(companyUsersPath(companyID), map[string]any{
		"name": "Dup", "email": email, "password": testPassword,
	}, ownerH)
	assert.Equal(t, 409, w.Code, w.Body.String())
}

func TestCompany_CreateUser_Success(t *testing.T) {
	ownerH, _, companyID := mustCompanyOwner(t)
	email := testhelper.UniqueEmail("kasir")
	before := countEvents(t, "company_member.added", companyID)

	w := testSrv.POST(companyUsersPath(companyID), map[string]any{
		"name": "Kasir Baru", "email": email, "password": testPassword,
	}, ownerH)
	require.Equal(t, 201, w.Code, w.Body.String())
	var got struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	testhelper.DecodeBody(t, w, &got)
	assert.NotEmpty(t, got.UserID)
	assert.Equal(t, email, got.Email)
	assert.Equal(t, "Kasir Baru", got.Name)
	assert.Equal(t, "active", got.Status)
	assert.Equal(t, before+1, countEvents(t, "company_member.added", companyID))

	// Akun baru langsung bisa login dan memilih company tersebut.
	tok := mustLogin(t, email, testConsoleSolution)
	require.NotNil(t, tok.CompanyID, "satu-satunya company langsung aktif")
	assert.Equal(t, companyID, *tok.CompanyID)
}

func TestCompany_CreateUser_ServiceToken(t *testing.T) {
	_, _, companyID := mustCompanyOwner(t)
	svc := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("svc")), "users:write"), ""))
	w := testSrv.POST(companyUsersPath(companyID), map[string]any{
		"name": "Dari Solution", "email": testhelper.UniqueEmail("svc"), "password": testPassword,
	}, svc)
	assert.Equal(t, 201, w.Code, w.Body.String())
}

// ── Permission platform ──────────────────────────────────────────────────────

func TestCompanyAdmin_CreatorGetsCompanyAdmin(t *testing.T) {
	f := newAdminFixture(t)
	w := testSrv.GET("/api/v1/companies/"+f.companyID+"/users/"+f.ownerID+"/effective-permissions?solution_id="+f.consoleSolutionID, f.ownerH)
	require.Equal(t, 200, w.Code, w.Body.String())
	var items []struct {
		Permission string `json:"permission"`
	}
	testhelper.DecodeBody(t, w, &items)
	got := map[string]bool{}
	for _, it := range items {
		got[it.Permission] = true
	}
	for _, p := range []string{"company.manage", "member.manage", "branch.manage", "role.manage", "role.assign"} {
		assert.True(t, got[p], "owner punya %s", p)
	}
}

func TestCompanyAdmin_StaffWithoutPermission_Forbidden(t *testing.T) {
	f := newAdminFixture(t)
	base := "/api/v1/companies/" + f.companyID
	branchID := mustCreateBranch(t, f.ownerH, f.companyID, testhelper.UniqueCode("BR"))

	// Baca tetap boleh untuk member aktif.
	assert.Equal(t, 200, testSrv.GET(base+"/members", f.staffH).Code)
	assert.Equal(t, 200, testSrv.GET(base+"/branches", f.staffH).Code)

	// Tulis ditolak tanpa permission platform.
	w := testSrv.PATCH(base, map[string]any{"name": "Diubah"}, f.staffH)
	require.Equal(t, 403, w.Code, "company.manage")
	assert.Contains(t, errorMessage(t, testhelper.DecodeError(t, w)), "do not have permission to manage this company")
	assert.Equal(t, 403, testSrv.POST(base+"/branches", map[string]any{"name": "B", "code": testhelper.UniqueCode("B")}, f.staffH).Code, "branch.manage")
	assert.Equal(t, 403, testSrv.PATCH("/api/v1/branches/"+branchID, map[string]any{"name": "X"}, f.staffH).Code, "branch.manage")
	assert.Equal(t, 403, testSrv.POST(base+"/members", map[string]any{"email": testhelper.UniqueEmail("x")}, f.staffH).Code, "member.manage")
	assert.Equal(t, 403, testSrv.POST(base+"/solutions/"+f.consoleSolutionID+"/roles", map[string]any{"code": "x", "name": "X"}, f.staffH).Code, "role.manage")
	assert.Equal(t, 403, testSrv.POST(assignPath(f.companyID, f.staffID), map[string]any{"role_id": f.companyAdminRoleID}, f.staffH).Code, "role.assign")

	// Owner (company_admin) boleh.
	assert.Equal(t, 200, testSrv.PATCH(base, map[string]any{"name": "Diubah Owner"}, f.ownerH).Code)
}

func TestCompanyAdmin_BranchScopedPermission(t *testing.T) {
	f := newAdminFixture(t)
	b1 := mustCreateBranch(t, f.ownerH, f.companyID, testhelper.UniqueCode("B1"))
	b2 := mustCreateBranch(t, f.ownerH, f.companyID, testhelper.UniqueCode("B2"))
	kacab := f.mustConsoleRole(t, "kepala_cabang", "branch.manage")
	f.assignStaff(t, kacab, "branch", b1)

	assert.Equal(t, 200, testSrv.PATCH("/api/v1/branches/"+b1, map[string]any{"name": "Cabang Satu"}, f.staffH).Code, "branch sendiri")
	assert.Equal(t, 403, testSrv.PATCH("/api/v1/branches/"+b2, map[string]any{"name": "X"}, f.staffH).Code, "branch lain")
	assert.Equal(t, 403, testSrv.POST("/api/v1/companies/"+f.companyID+"/branches",
		map[string]any{"name": "Baru", "code": testhelper.UniqueCode("B")}, f.staffH).Code, "tambah branch butuh scope company")
}

func TestCompanyAdmin_NoPrivilegeEscalation(t *testing.T) {
	f := newAdminFixture(t)
	assigner := f.mustConsoleRole(t, "pemberi_role", "role.assign", "role.manage")
	f.assignStaff(t, assigner, "company", f.companyID)

	// Tidak bisa memberi dirinya role platform (company_admin) atau membuat role console baru.
	w := testSrv.POST(assignPath(f.companyID, f.staffID), map[string]any{"role_id": f.companyAdminRoleID}, f.staffH)
	assert.Equal(t, 403, w.Code, w.Body.String())
	w = testSrv.POST("/api/v1/companies/"+f.companyID+"/solutions/"+f.consoleSolutionID+"/roles",
		map[string]any{"code": "super", "name": "Super"}, f.staffH)
	assert.Equal(t, 403, w.Code, w.Body.String())

	// Role solution produk tetap bisa dikelola dan di-assign.
	solutionID := mustCreateSolution(t, testhelper.UniqueCode("app"))
	w = testSrv.POST("/api/v1/companies/"+f.companyID+"/solutions/"+solutionID+"/roles",
		map[string]any{"code": "kasir", "name": "Kasir"}, f.staffH)
	require.Equal(t, 201, w.Code, w.Body.String())
	var role struct {
		ID string `json:"id"`
	}
	testhelper.DecodeBody(t, w, &role)
	w = testSrv.POST(assignPath(f.companyID, f.ownerID), map[string]any{"role_id": role.ID}, f.staffH)
	assert.Equal(t, 201, w.Code, w.Body.String())
}

func TestCompanyAdmin_ServiceTokenBypassesPlatformPermission(t *testing.T) {
	f := newAdminFixture(t)
	svc := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("svc")), "org:write"), ""))
	w := testSrv.POST("/api/v1/companies/"+f.companyID+"/branches",
		map[string]any{"name": "Dari BE", "code": testhelper.UniqueCode("BE")}, svc)
	assert.Equal(t, 201, w.Code, w.Body.String())
}
