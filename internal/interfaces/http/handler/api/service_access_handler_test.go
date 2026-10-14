package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/testhelper"
)

// Akses token service ke endpoint tenant & organisasi (spec §2.8): hanya untuk tenant
// yang subscribe solution client, dan sesuai scope.

func TestServiceAccess_TenantEndpoints(t *testing.T) {
	f := newSubFixture(t)
	mustSubscribe(t, f.tenantID, f.basicPlan)
	reader := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "users:read"), ""))
	writer := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "users:write"), ""))

	assert.Equal(t, 200, testSrv.GET("/api/v1/tenants/"+f.tenantID, reader).Code)
	assert.Equal(t, 200, testSrv.GET(membersPath(f.tenantID), reader).Code)
	assert.Equal(t, 403, testSrv.POST(membersPath(f.tenantID), map[string]any{"email": "x@test.local"}, reader).Code, "users:read tidak boleh menulis")

	userID := mustCreateUser(t, testhelper.UniqueEmail("svc"))
	w := testSrv.POST(membersPath(f.tenantID), map[string]any{"user_id": userID}, writer)
	assert.Equal(t, 201, w.Code, w.Body.String())

	// Membuat & mendaftar tenant tetap khusus token user.
	assert.Equal(t, 403, testSrv.GET("/api/v1/tenants", reader).Code)
}

func TestServiceAccess_TenantNotSubscribed(t *testing.T) {
	f := newSubFixture(t) // tenant belum subscribe
	reader := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "users:read", "org:read"), ""))
	assert.Equal(t, 403, testSrv.GET("/api/v1/tenants/"+f.tenantID, reader).Code)
	assert.Equal(t, 403, testSrv.GET(companiesPath(f.tenantID), reader).Code)

	// Subscription dibatalkan → akses service hilang.
	subID := mustSubscribe(t, f.tenantID, f.basicPlan)
	assert.Equal(t, 200, testSrv.GET("/api/v1/tenants/"+f.tenantID, reader).Code)
	require.Equal(t, 200, testSrv.POST(subscriptionPath(subID, "cancel"), nil, adminHeader()).Code)
	assert.Equal(t, 403, testSrv.GET("/api/v1/tenants/"+f.tenantID, reader).Code)
}

func TestServiceAccess_OrganizationEndpoints(t *testing.T) {
	f := newSubFixture(t)
	mustSubscribe(t, f.tenantID, f.basicPlan)
	writer := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "org:read", "org:write"), ""))
	readerOnly := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, f.solutionID, "org:read"), ""))

	companyID := mustCreateCompany(t, writer, f.tenantID, "HQ")
	assert.Equal(t, 201, testSrv.POST(branchesPath(companyID), map[string]any{"name": "B1", "code": "B1"}, writer).Code)
	assert.Equal(t, 200, testSrv.GET(branchesPath(companyID), readerOnly).Code)
	assert.Equal(t, 403, testSrv.POST(companiesPath(f.tenantID), map[string]any{"name": "X", "code": "X"}, readerOnly).Code)

	// Client solution lain tidak bisa melihat company tenant ini (404, tidak membocorkan).
	other := newSubFixture(t)
	foreign := testhelper.BearerHeader(mustServiceToken(t, mustCreateClient(t, other.solutionID, "org:read"), ""))
	assert.Equal(t, 404, testSrv.GET(branchesPath(companyID), foreign).Code)
}
