package entity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/solutionclient/constant"
	"platform-service/internal/domain/solutionclient/entity"
	"platform-service/internal/domain/solutionclient/valueobject"
	"platform-service/internal/testhelper"
)

func newClient(t *testing.T, scopes ...string) *entity.SolutionClient {
	t.Helper()
	c, err := entity.NewSolutionClient(entity.NewSolutionClientParams{
		SolutionID: "sol", Name: " Easy HR Backend ", ClientID: "cli_x", ClientSecretHash: "h1", Scopes: scopes,
	}, time.Now())
	require.NoError(t, err)
	return c
}

func TestNewSolutionClient(t *testing.T) {
	c := newClient(t, "usage:write", "authz:check", "usage:write")
	assert.Equal(t, "Easy HR Backend", c.Name)
	assert.Equal(t, []string{"authz:check", "usage:write"}, c.Scopes.Values())
	assert.True(t, c.IsActive())
	assert.True(t, c.SecretMatches("h1"))
	assert.False(t, c.SecretMatches("h2"))
}

func TestNewSolutionClient_Invalid(t *testing.T) {
	_, err := entity.NewSolutionClient(entity.NewSolutionClientParams{SolutionID: "s", Name: "X", ClientID: "c", ClientSecretHash: "h", Scopes: []string{"admin:all"}}, time.Now())
	testhelper.AssertDomainError(t, err, constant.CodeClientScopeInvalid)

	_, err = entity.NewSolutionClient(entity.NewSolutionClientParams{SolutionID: "s", Name: " ", ClientID: "c", ClientSecretHash: "h"}, time.Now())
	testhelper.AssertDomainError(t, err, constant.CodeClientNameInvalid)
}

func TestSolutionClient_GrantableScopes(t *testing.T) {
	c := newClient(t, "usage:write", "authz:check")

	all, err := c.GrantableScopes(valueobject.Scopes{})
	require.NoError(t, err)
	assert.Equal(t, []string{"authz:check", "usage:write"}, all.Values())

	req, err := valueobject.ParseScopeParam("usage:write")
	require.NoError(t, err)
	sub, err := c.GrantableScopes(req)
	require.NoError(t, err)
	assert.Equal(t, []string{"usage:write"}, sub.Values())

	req, err = valueobject.ParseScopeParam("usage:write catalog:sync")
	require.NoError(t, err)
	_, err = c.GrantableScopes(req)
	testhelper.AssertDomainError(t, err, constant.CodeClientScopeInvalid)
}

func TestSolutionClient_RotateAndRevoke(t *testing.T) {
	c := newClient(t)
	require.NoError(t, c.RotateSecret("h2", time.Now()))
	assert.True(t, c.SecretMatches("h2"))
	assert.False(t, c.SecretMatches("h1"))

	c.Revoke(time.Now())
	assert.False(t, c.IsActive())
	require.NotNil(t, c.RevokedAt)
	testhelper.AssertDomainError(t, c.RotateSecret("h3", time.Now()), constant.CodeClientRevoked)
}

func TestScopes_Intersect(t *testing.T) {
	a, err := valueobject.NewScopes([]string{"usage:write", "authz:check", "events:read"})
	require.NoError(t, err)
	b, err := valueobject.NewScopes([]string{"events:read", "usage:write"})
	require.NoError(t, err)
	assert.Equal(t, []string{"events:read", "usage:write"}, a.Intersect(b).Values())
}
