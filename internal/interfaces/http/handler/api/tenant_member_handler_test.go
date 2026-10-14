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

func membersPath(tenantID string) string {
	return "/api/v1/tenants/" + tenantID + "/members"
}

type memberData struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Status   string `json:"status"`
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestTenantMember_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(membersPath(uuid.NewString())).Code)
}

func TestTenantMember_List_Forbidden_NonMember(t *testing.T) {
	_, _, tenantID := mustTenantOwner(t)
	_, strangerID, _ := mustTenantOwner(t)
	assert.Equal(t, 403, testSrv.GET(membersPath(tenantID), tenantToken(strangerID, tenantID)).Code)
}

func TestTenantMember_List_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.GET(membersPath(uuid.NewString()), adminHeader()).Code)
}

func TestTenantMember_List_Success(t *testing.T) {
	h, ownerID, tenantID := mustTenantOwner(t)

	w := testSrv.GET(membersPath(tenantID), h)
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

func TestTenantMember_Add_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(membersPath(uuid.NewString()), map[string]any{"email": "a@b.co"}).Code)
}

func TestTenantMember_Add_Forbidden_TenantNotSelected(t *testing.T) {
	tok, _, _ := mustLoginNewUser(t)
	h := testhelper.BearerHeader(tok.AccessToken)
	tenantID := mustCreateTenant(t, h)

	w := testSrv.POST(membersPath(tenantID), map[string]any{"email": testhelper.UniqueEmail("x")}, h)
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestTenantMember_Add_InvalidPayload(t *testing.T) {
	h, _, tenantID := mustTenantOwner(t)
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
			w := testSrv.POST(membersPath(tenantID), body, h)
			assert.Equal(t, 422, w.Code, w.Body.String())
		})
	}
}

func TestTenantMember_Add_UserNotFound(t *testing.T) {
	h, _, tenantID := mustTenantOwner(t)
	w := testSrv.POST(membersPath(tenantID), map[string]any{"email": testhelper.UniqueEmail("ghost")}, h)
	require.Equal(t, 422, w.Code, w.Body.String())
	assert.Equal(t, "User to add was not found.", errorMessage(t, testhelper.DecodeError(t, w)))
}

func TestTenantMember_Add_TenantNotFound(t *testing.T) {
	w := testSrv.POST(membersPath(uuid.NewString()), map[string]any{"email": testhelper.UniqueEmail("x")}, adminHeader())
	assert.Equal(t, 404, w.Code, w.Body.String())
}

func TestTenantMember_Add_Duplicate(t *testing.T) {
	h, ownerID, tenantID := mustTenantOwner(t)
	w := testSrv.POST(membersPath(tenantID), map[string]any{"user_id": ownerID}, h)
	assert.Equal(t, 409, w.Code, w.Body.String())
}

func TestTenantMember_Add_Success_ByEmail(t *testing.T) {
	h, _, tenantID := mustTenantOwner(t)
	email := testhelper.UniqueEmail("newmember")
	userID := mustCreateUser(t, email)

	w := testSrv.POST(membersPath(tenantID), map[string]any{"email": upper(email), "status": "invited"}, h)
	require.Equal(t, 201, w.Code, w.Body.String())
	var got memberData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, tenantID, got.TenantID)
	assert.Equal(t, userID, got.UserID)
	assert.Equal(t, lower(email), got.Email)
	assert.Equal(t, "invited", got.Status)

	// Owner (create) + member baru.
	assert.Equal(t, 2, countEvents(t, "tenant_member.added", tenantID))
}

func TestTenantMember_Add_Success_PlatformAdmin(t *testing.T) {
	tenantID := mustCreateTenant(t, adminHeader())
	userID := mustCreateUser(t, testhelper.UniqueEmail("byadmin"))

	w := testSrv.POST(membersPath(tenantID), map[string]any{"user_id": userID}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())

	// Member baru langsung bisa mengakses tenant (dengan tid).
	assert.Equal(t, 200, testSrv.GET("/api/v1/tenants/"+tenantID, tenantToken(userID, tenantID)).Code)
}

func TestTenantMember_Add_RejoinInactiveMember(t *testing.T) {
	h, _, tenantID := mustTenantOwner(t)
	userID := mustCreateUser(t, testhelper.UniqueEmail("rejoin"))
	require.Equal(t, 201, testSrv.POST(membersPath(tenantID), map[string]any{"user_id": userID}, h).Code)
	_, err := testSrv.DB.ExecContext(context.Background(),
		`UPDATE tenant_members SET status = 'inactive' WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID)
	require.NoError(t, err)

	w := testSrv.POST(membersPath(tenantID), map[string]any{"user_id": userID}, h)
	require.Equal(t, 201, w.Code, w.Body.String())
	var got memberData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "active", got.Status)
}
