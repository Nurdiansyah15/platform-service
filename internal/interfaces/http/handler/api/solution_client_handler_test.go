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

func clientsPath(solutionID string) string {
	return "/api/v1/solutions/" + solutionID + "/clients"
}

func rotatePath(id string) string {
	return "/api/v1/solution-clients/" + id + "/rotate-secret"
}

// ── Create ───────────────────────────────────────────────────────────────────

func TestSolutionClient_Create_Unauthenticated(t *testing.T) {
	w := testSrv.POST(clientsPath(uuid.NewString()), map[string]any{"name": "X"})
	assert.Equal(t, 401, w.Code)
}

func TestSolutionClient_Create_Forbidden_NonAdmin(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cl"))
	tok, _, _ := mustLoginNewUser(t)
	w := testSrv.POST(clientsPath(solID), map[string]any{"name": "X"}, testhelper.BearerHeader(tok.AccessToken))
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestSolutionClient_Create_InvalidPayload(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cl"))

	w := testSrv.POST(clientsPath(solID), map[string]any{"scopes": []string{"usage:write"}}, adminHeader())
	assert.Equal(t, 422, w.Code, "tanpa name: %s", w.Body.String())

	w = testSrv.POST(clientsPath(solID), map[string]any{"name": "X", "scopes": []string{"admin:everything"}}, adminHeader())
	require.Equal(t, 422, w.Code, "scope tak dikenal: %s", w.Body.String())
	assert.Equal(t, "One or more scopes are not recognized.", errorMessage(t, testhelper.DecodeError(t, w)))

	w = testSrv.POST(clientsPath("bukan-uuid"), map[string]any{"name": "X"}, adminHeader())
	assert.Equal(t, 422, w.Code, "solutionId bukan uuid: %s", w.Body.String())
}

func TestSolutionClient_Create_SolutionNotFound(t *testing.T) {
	w := testSrv.POST(clientsPath(uuid.NewString()), map[string]any{"name": "X"}, adminHeader())
	assert.Equal(t, 404, w.Code, w.Body.String())
}

func TestSolutionClient_Create_Success(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cl"))

	w := testSrv.POST(clientsPath(solID), map[string]any{
		"name":   "Easy HR Backend Prod",
		"scopes": []string{"usage:write", "authz:check", "usage:write"},
	}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())

	var got clientCredentials
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, solID, got.SolutionID)
	assert.Regexp(t, `^cli_[0-9a-f]{24}$`, got.ClientID)
	assert.Len(t, got.ClientSecret, 43, "256 bit base64url")
	assert.Equal(t, []string{"authz:check", "usage:write"}, got.Scopes)
	assert.Equal(t, "active", got.Status)

	var hash string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT client_secret_hash FROM solution_clients WHERE id = $1`, got.ID).Scan(&hash))
	assert.NotEqual(t, got.ClientSecret, hash, "secret mentah tidak boleh disimpan")
	assert.Len(t, hash, 64)
}

// ── Rotate secret ────────────────────────────────────────────────────────────

func TestSolutionClient_RotateSecret_Unauthenticated(t *testing.T) {
	w := testSrv.POST(rotatePath(uuid.NewString()), nil)
	assert.Equal(t, 401, w.Code)
}

func TestSolutionClient_RotateSecret_Forbidden_NonAdmin(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("rot")))
	tok, _, _ := mustLoginNewUser(t)
	w := testSrv.POST(rotatePath(c.ID), nil, testhelper.BearerHeader(tok.AccessToken))
	assert.Equal(t, 403, w.Code, w.Body.String())
}

func TestSolutionClient_RotateSecret_InvalidID(t *testing.T) {
	w := testSrv.POST(rotatePath("bukan-uuid"), nil, adminHeader())
	assert.Equal(t, 422, w.Code, w.Body.String())
}

func TestSolutionClient_RotateSecret_NotFound(t *testing.T) {
	w := testSrv.POST(rotatePath(uuid.NewString()), nil, adminHeader())
	assert.Equal(t, 404, w.Code, w.Body.String())
}

func TestSolutionClient_RotateSecret_RevokedConflict(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("rot")))
	_, err := testSrv.DB.ExecContext(context.Background(),
		`UPDATE solution_clients SET status = 'revoked', revoked_at = NOW() WHERE id = $1`, c.ID)
	require.NoError(t, err)

	w := testSrv.POST(rotatePath(c.ID), nil, adminHeader())
	assert.Equal(t, 409, w.Code, w.Body.String())
}

func TestSolutionClient_RotateSecret_Success(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("rot")), "usage:write")
	oldToken := mustServiceToken(t, c, "")

	w := testSrv.POST(rotatePath(c.ID), nil, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var rotated clientCredentials
	testhelper.DecodeBody(t, w, &rotated)
	assert.Equal(t, c.ClientID, rotated.ClientID, "client_id tidak berubah")
	assert.NotEqual(t, c.ClientSecret, rotated.ClientSecret)

	// Secret lama tidak bisa menukar token lagi.
	w = testSrv.POSTForm("/api/v1/oauth/token", oauthForm(c.ClientID, c.ClientSecret))
	assert.Equal(t, 401, w.Code, w.Body.String())

	// Secret baru bisa.
	mustServiceToken(t, rotated, "")

	// Token yang sudah terbit sebelum rotasi tetap berlaku sampai exp.
	w = testSrv.GET("/api/v1/solutions", testhelper.BearerHeader(oldToken))
	assert.Equal(t, 200, w.Code, w.Body.String())
}

// ── List / Get / Update / Revoke ─────────────────────────────────────────────

func clientPath(id string) string {
	return "/api/v1/solution-clients/" + id
}

func TestSolutionClient_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(clientsPath(uuid.NewString())).Code)
}

func TestSolutionClient_List_Forbidden_NonAdmin(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cl"))
	tok, _, _ := mustLoginNewUser(t)
	assert.Equal(t, 403, testSrv.GET(clientsPath(solID), testhelper.BearerHeader(tok.AccessToken)).Code)
}

func TestSolutionClient_List_InvalidQuery(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cl"))
	assert.Equal(t, 422, testSrv.GET(clientsPath(solID)+"?status=deleted", adminHeader()).Code)
}

func TestSolutionClient_List_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.GET(clientsPath(uuid.NewString()), adminHeader()).Code)
}

func TestSolutionClient_List_Success(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cl"))
	active := mustCreateClient(t, solID, "usage:write")
	revoked := mustCreateClient(t, solID)
	require.Equal(t, 200, testSrv.POST(clientPath(revoked.ID)+"/revoke", nil, adminHeader()).Code)
	mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("cl")))

	w := testSrv.GET(clientsPath(solID), adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp struct {
		Data []map[string]any `json:"data"`
		Meta struct {
			Total int64 `json:"total"`
		} `json:"meta"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, int64(2), resp.Meta.Total, "hanya client solution ini")
	require.Len(t, resp.Data, 2)
	for _, it := range resp.Data {
		assert.NotContains(t, it, "client_secret")
	}

	w = testSrv.GET(clientsPath(solID)+"?status=active", adminHeader())
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, active.ID, resp.Data[0]["id"])
	assert.Equal(t, []any{"usage:write"}, resp.Data[0]["scopes"])
}

func TestSolutionClient_Get_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(clientPath(uuid.NewString())).Code)
}

func TestSolutionClient_Get_Forbidden_NonAdmin(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("cl")))
	tok, _, _ := mustLoginNewUser(t)
	assert.Equal(t, 403, testSrv.GET(clientPath(c.ID), testhelper.BearerHeader(tok.AccessToken)).Code)
}

func TestSolutionClient_Get_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.GET(clientPath(uuid.NewString()), adminHeader()).Code)
	assert.Equal(t, 422, testSrv.GET(clientPath("bukan-uuid"), adminHeader()).Code)
}

func TestSolutionClient_Get_Success(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("cl")), "authz:check")
	w := testSrv.GET(clientPath(c.ID), adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got clientCredentials
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, c.ClientID, got.ClientID)
	assert.Equal(t, []string{"authz:check"}, got.Scopes)
	assert.Empty(t, got.ClientSecret, "secret tidak pernah ditampilkan lagi")
}

func TestSolutionClient_Update_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.PATCH(clientPath(uuid.NewString()), map[string]any{"name": "X"}).Code)
}

func TestSolutionClient_Update_Forbidden_NonAdmin(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("cl")))
	tok, _, _ := mustLoginNewUser(t)
	assert.Equal(t, 403, testSrv.PATCH(clientPath(c.ID), map[string]any{"name": "X"}, testhelper.BearerHeader(tok.AccessToken)).Code)
}

func TestSolutionClient_Update_InvalidPayload(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("cl")))
	assert.Equal(t, 422, testSrv.PATCH(clientPath(c.ID), map[string]any{"name": "  "}, adminHeader()).Code)
	assert.Equal(t, 422, testSrv.PATCH(clientPath(c.ID), map[string]any{"scopes": []string{"admin:everything"}}, adminHeader()).Code)
}

func TestSolutionClient_Update_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.PATCH(clientPath(uuid.NewString()), map[string]any{"name": "X"}, adminHeader()).Code)
}

func TestSolutionClient_Update_RevokedConflict(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("cl")))
	require.Equal(t, 200, testSrv.POST(clientPath(c.ID)+"/revoke", nil, adminHeader()).Code)
	assert.Equal(t, 409, testSrv.PATCH(clientPath(c.ID), map[string]any{"name": "X"}, adminHeader()).Code)
}

func TestSolutionClient_Update_Success(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("cl"))
	c := mustCreateClient(t, solID, "org:read")
	companyID := mustCreateCompany(t, adminHeader())
	tok := testhelper.BearerHeader(mustServiceToken(t, c, ""))
	require.Equal(t, 200, testSrv.GET(branchesPath(companyID), tok).Code)

	w := testSrv.PATCH(clientPath(c.ID), map[string]any{"name": "Renamed", "scopes": []string{"usage:write"}}, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "Renamed", got.Name)
	assert.Equal(t, []string{"usage:write"}, got.Scopes)

	// Scope org:read dicabut → token lama langsung kehilangan akses.
	assert.Equal(t, 403, testSrv.GET(branchesPath(companyID), tok).Code)
}

func TestSolutionClient_Revoke_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(clientPath(uuid.NewString())+"/revoke", nil).Code)
}

func TestSolutionClient_Revoke_Forbidden_NonAdmin(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("cl")))
	tok, _, _ := mustLoginNewUser(t)
	assert.Equal(t, 403, testSrv.POST(clientPath(c.ID)+"/revoke", nil, testhelper.BearerHeader(tok.AccessToken)).Code)
}

func TestSolutionClient_Revoke_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.POST(clientPath(uuid.NewString())+"/revoke", nil, adminHeader()).Code)
}

func TestSolutionClient_Revoke_Success(t *testing.T) {
	c := mustCreateClient(t, mustCreateSolution(t, testhelper.UniqueCode("cl")), "usage:write")
	tok := testhelper.BearerHeader(mustServiceToken(t, c, ""))
	require.Equal(t, 200, testSrv.GET("/api/v1/solutions", tok).Code)

	w := testSrv.POST(clientPath(c.ID)+"/revoke", nil, adminHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got struct {
		Status    string  `json:"status"`
		RevokedAt *string `json:"revoked_at"`
	}
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "revoked", got.Status)
	assert.NotNil(t, got.RevokedAt)

	// Token yang sudah terbit langsung ditolak, dan kredensial tidak bisa menukar token lagi.
	assert.Equal(t, 401, testSrv.GET("/api/v1/solutions", tok).Code)
	assert.Equal(t, 401, testSrv.POSTForm("/api/v1/oauth/token", oauthForm(c.ClientID, c.ClientSecret)).Code)

	// Idempotent.
	assert.Equal(t, 200, testSrv.POST(clientPath(c.ID)+"/revoke", nil, adminHeader()).Code)
}
