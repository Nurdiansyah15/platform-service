package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/testhelper"
)

// Akses token service ke endpoint company & organisasi (spec §2.8): hanya untuk company
// yang subscribe solution client, dan sesuai scope.

func TestServiceAccess_CompanyEndpoints(t *testing.T) {
	f := newSubFixture(t)
	mustSubscribe(t, f.companyID, f.basicPlan)
	reader := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "users:read"), ""))
	writer := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "users:write"), ""))

	assert.Equal(t, 200, testSrv.GET("/api/v1/companies/"+f.companyID, reader).Code)
	assert.Equal(t, 200, testSrv.GET(membersPath(f.companyID), reader).Code)
	assert.Equal(t, 403, testSrv.POST(membersPath(f.companyID), map[string]any{"email": "x@test.local"}, reader).Code, "users:read tidak boleh menulis")

	userID := mustCreateUser(t, testhelper.UniqueEmail("svc"))
	w := testSrv.POST(membersPath(f.companyID), map[string]any{"user_id": userID}, writer)
	assert.Equal(t, 201, w.Code, w.Body.String())

	// Membuat & mendaftar company tetap khusus token user.
	assert.Equal(t, 403, testSrv.GET("/api/v1/companies", reader).Code)
}

func TestServiceAccess_CompanyNotSubscribed(t *testing.T) {
	f := newSubFixture(t) // company belum subscribe
	reader := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "users:read", "org:read"), ""))
	assert.Equal(t, 403, testSrv.GET("/api/v1/companies/"+f.companyID, reader).Code)
	assert.Equal(t, 403, testSrv.GET(branchesPath(f.companyID), reader).Code)

	// Subscription dibatalkan → akses service hilang.
	subID := mustSubscribe(t, f.companyID, f.basicPlan)
	assert.Equal(t, 200, testSrv.GET("/api/v1/companies/"+f.companyID, reader).Code)
	require.Equal(t, 200, testSrv.POST(subscriptionPath(subID, "cancel"), nil, adminHeader()).Code)
	assert.Equal(t, 403, testSrv.GET("/api/v1/companies/"+f.companyID, reader).Code)
}

func TestServiceAccess_OrganizationEndpoints(t *testing.T) {
	f := newSubFixture(t)
	mustSubscribe(t, f.companyID, f.basicPlan)
	writer := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "org:read", "org:write"), ""))
	readerOnly := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "org:read"), ""))

	assert.Equal(t, 201, testSrv.POST(branchesPath(f.companyID), map[string]any{"name": "B1", "code": "B1"}, writer).Code)
	assert.Equal(t, 200, testSrv.GET(branchesPath(f.companyID), readerOnly).Code)
	// Membuat company tetap khusus token user, bukan token service.
	assert.Equal(t, 403, testSrv.POST("/api/v1/companies", map[string]any{"name": "X", "slug": "x"}, readerOnly).Code)

	// Client solution lain tidak bisa melihat company ini (tidak subscribe → 403).
	other := newSubFixture(t)
	foreign := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, other.solutionID, "org:read"), ""))
	assert.Equal(t, 403, testSrv.GET(branchesPath(f.companyID), foreign).Code)
}
