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

func membersPath(companyID string) string {
	return "/api/v1/companies/" + companyID + "/members"
}

type memberData struct {
	CompanyID string `json:"company_id"`
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Status    string `json:"status"`
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestCompanyMember_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(membersPath(uuid.NewString())).Code)
}

func TestCompanyMember_List_Forbidden_NonMember(t *testing.T) {
	_, _, companyID := mustCompanyOwner(t)
	_, strangerID, _ := mustCompanyOwner(t)
	assert.Equal(t, 403, testSrv.GET(membersPath(companyID), companyToken(strangerID, companyID)).Code)
}

func TestCompanyMember_List_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.GET(membersPath(uuid.NewString()), adminHeader()).Code)
}

func TestCompanyMember_List_Success(t *testing.T) {
	h, ownerID, companyID := mustCompanyOwner(t)

	w := testSrv.GET(membersPath(companyID), h)
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp struct {
		Data []memberData `json:"data"`
		Meta struct {
			Total int64 `json:"total"`
		} `json:"meta"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, ownerID, resp.Data[0].UserID)
	assert.Equal(t, "active", resp.Data[0].Status)
	assert.NotEmpty(t, resp.Data[0].Email)
	assert.Equal(t, int64(1), resp.Meta.Total)
}

// ── Add ──────────────────────────────────────────────────────────────────────

func TestCompanyMember_Add_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(membersPath(uuid.NewString()), map[string]any{"email": "a@b.co"}).Code)
}

func TestCompanyMember_Add_Forbidden_CompanyNotSelected(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)
	h := testhelper.BearerHeader(tok.AccessToken)
	companyID := mustCreateCompany(t, h)

	w := testSrv.POST(membersPath(companyID), map[string]any{"email": testhelper.UniqueEmail("x")}, h)
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestCompanyMember_Add_InvalidPayload(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	email := testhelper.UniqueEmail("both")
	userID := mustCreateUser(t, email)

	tests := map[string]map[string]any{
		"tanpa identitas":    {},
		"user_id dan email":  {"user_id": userID, "email": email},
		"user_id bukan uuid": {"user_id": "abc"},
		"status inactive":    {"email": email, "status": "inactive"},
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			w := testSrv.POST(membersPath(companyID), body, h)
			assert.Equal(t, 422, w.Code, w.Body.String())
		})
	}
}

func TestCompanyMember_Add_UserNotFound(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	w := testSrv.POST(membersPath(companyID), map[string]any{"email": testhelper.UniqueEmail("ghost")}, h)
	require.Equal(t, 422, w.Code, w.Body.String())
	assert.Equal(t, "User to add was not found.", errorMessage(t, testhelper.DecodeError(t, w)))
}

func TestCompanyMember_Add_CompanyNotFound(t *testing.T) {
	w := testSrv.POST(membersPath(uuid.NewString()), map[string]any{"email": testhelper.UniqueEmail("x")}, adminHeader())
	assert.Equal(t, 404, w.Code, w.Body.String())
}

func TestCompanyMember_Add_Duplicate(t *testing.T) {
	h, ownerID, companyID := mustCompanyOwner(t)
	w := testSrv.POST(membersPath(companyID), map[string]any{"user_id": ownerID}, h)
	assert.Equal(t, 409, w.Code, w.Body.String())
}

func TestCompanyMember_Add_Success_ByEmail(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	email := testhelper.UniqueEmail("newmember")
	userID := mustCreateUser(t, email)

	w := testSrv.POST(membersPath(companyID), map[string]any{"email": upper(email), "status": "invited"}, h)
	require.Equal(t, 201, w.Code, w.Body.String())
	var got memberData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, companyID, got.CompanyID)
	assert.Equal(t, userID, got.UserID)
	assert.Equal(t, lower(email), got.Email)
	assert.Equal(t, "invited", got.Status)

	// Owner (create) + member baru.
	assert.Equal(t, 2, countEvents(t, "company_member.added", companyID))
}

func TestCompanyMember_Add_Success_PlatformAdmin(t *testing.T) {
	companyID := mustCreateCompany(t, adminHeader())
	userID := mustCreateUser(t, testhelper.UniqueEmail("byadmin"))

	w := testSrv.POST(membersPath(companyID), map[string]any{"user_id": userID}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())

	// Member baru langsung bisa mengakses company (dengan tid).
	assert.Equal(t, 200, testSrv.GET("/api/v1/companies/"+companyID, companyToken(userID, companyID)).Code)
}

func TestCompanyMember_Add_RejoinInactiveMember(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	userID := mustCreateUser(t, testhelper.UniqueEmail("rejoin"))
	require.Equal(t, 201, testSrv.POST(membersPath(companyID), map[string]any{"user_id": userID}, h).Code)
	_, err := testSrv.DB.ExecContext(context.Background(),
		`UPDATE company_members SET status = 'inactive' WHERE company_id = $1 AND user_id = $2`, companyID, userID)
	require.NoError(t, err)

	w := testSrv.POST(membersPath(companyID), map[string]any{"user_id": userID}, h)
	require.Equal(t, 201, w.Code, w.Body.String())
	var got memberData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "active", got.Status)
}

// ── Update / Remove ──────────────────────────────────────────────────────────

func memberPath(companyID, userID string) string {
	return membersPath(companyID) + "/" + userID
}

func TestCompanyMember_Update_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.PATCH(memberPath(uuid.NewString(), uuid.NewString()), map[string]any{"status": "active"}).Code)
}

func TestCompanyMember_Update_Forbidden_NonMember(t *testing.T) {
	_, ownerID, companyID := mustCompanyOwner(t)
	_, strangerID, _ := mustCompanyOwner(t)
	w := testSrv.PATCH(memberPath(companyID, ownerID), map[string]any{"status": "invited"}, companyToken(strangerID, companyID))
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestCompanyMember_Update_InvalidPayload(t *testing.T) {
	h, ownerID, companyID := mustCompanyOwner(t)
	assert.Equal(t, 422, testSrv.PATCH(memberPath(companyID, ownerID), map[string]any{"status": "banned"}, h).Code)
	assert.Equal(t, 422, testSrv.PATCH(memberPath(companyID, ownerID), map[string]any{}, h).Code)
	assert.Equal(t, 422, testSrv.PATCH(memberPath(companyID, "bukan-uuid"), map[string]any{"status": "active"}, h).Code)
}

func TestCompanyMember_Update_NotFound(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	notMember := mustCreateUser(t, testhelper.UniqueEmail("nm"))
	assert.Equal(t, 404, testSrv.PATCH(memberPath(companyID, notMember), map[string]any{"status": "active"}, h).Code)
	assert.Equal(t, 404, testSrv.PATCH(memberPath(uuid.NewString(), notMember), map[string]any{"status": "active"}, adminHeader()).Code)
}

func TestCompanyMember_Update_Success(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	userID := mustCreateUser(t, testhelper.UniqueEmail("m"))
	require.Equal(t, 201, testSrv.POST(membersPath(companyID), map[string]any{"user_id": userID}, h).Code)

	w := testSrv.PATCH(memberPath(companyID, userID), map[string]any{"status": "invited"}, h)
	require.Equal(t, 200, w.Code, w.Body.String())
	var got memberData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, userID, got.UserID)
	assert.Equal(t, "invited", got.Status)
	assert.NotEmpty(t, got.Email)
	assert.Equal(t, 1, countEvents(t, "company_member.updated", companyID))

	// Status sama → tidak ada event baru.
	require.Equal(t, 200, testSrv.PATCH(memberPath(companyID, userID), map[string]any{"status": "invited"}, h).Code)
	assert.Equal(t, 1, countEvents(t, "company_member.updated", companyID))
}

func TestCompanyMember_Remove_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.DELETE(memberPath(uuid.NewString(), uuid.NewString())).Code)
}

func TestCompanyMember_Remove_Forbidden_NonMember(t *testing.T) {
	_, ownerID, companyID := mustCompanyOwner(t)
	_, strangerID, _ := mustCompanyOwner(t)
	assert.Equal(t, 403, testSrv.DELETE(memberPath(companyID, ownerID), companyToken(strangerID, companyID)).Code)
}

func TestCompanyMember_Remove_NotFound(t *testing.T) {
	h, _, companyID := mustCompanyOwner(t)
	assert.Equal(t, 404, testSrv.DELETE(memberPath(companyID, uuid.NewString()), h).Code)
}

func TestCompanyMember_Remove_Success(t *testing.T) {
	f := newAuthzFixture(t)
	f.assign(t, f.hrAdmin, "", "")

	w := testSrv.DELETE(memberPath(f.companyID, f.staffID), f.ownerH)
	require.Equal(t, 200, w.Code, w.Body.String())

	var status string
	var assignments int
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT status FROM company_members WHERE company_id = $1 AND user_id = $2`, f.companyID, f.staffID).Scan(&status))
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM user_role_assignments WHERE company_id = $1 AND user_id = $2`, f.companyID, f.staffID).Scan(&assignments))
	assert.Equal(t, "inactive", status)
	assert.Equal(t, 0, assignments, "role assignment ikut dihapus")
	assert.Equal(t, 1, countEvents(t, "company_member.removed", f.companyID))
	assert.False(t, f.serviceCheck(t, "employee.read", "", "").Allowed)

	// Idempotent.
	assert.Equal(t, 200, testSrv.DELETE(memberPath(f.companyID, f.staffID), f.ownerH).Code)
	assert.Equal(t, 1, countEvents(t, "company_member.removed", f.companyID))
}
