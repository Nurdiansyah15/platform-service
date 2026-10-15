package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	permentity "platform-service/internal/domain/permission/entity"
	"platform-service/internal/domain/role/constant"
	"platform-service/internal/domain/role/entity"
	"platform-service/internal/domain/role/service"
	"platform-service/internal/testhelper"
)

func perm(t *testing.T, code string, deprecated bool) *permentity.Permission {
	t.Helper()
	d, err := permentity.NewDefinition(code, code, nil)
	require.NoError(t, err)
	p := permentity.NewPermission("s", d, time.Now())
	if deprecated {
		p.Deprecate(time.Now())
	}
	return p
}

func TestResolvePermissions(t *testing.T) {
	read, write, legacy := perm(t, "employee.read", false), perm(t, "employee.write", false), perm(t, "legacy.export", true)
	catalog := []*permentity.Permission{read, write, legacy}

	got, err := service.ResolvePermissions([]string{"employee.read", "employee.write"}, catalog, nil)
	require.NoError(t, err)
	assert.Len(t, got, 2)

	_, err = service.ResolvePermissions([]string{"employee.read", "employee.read"}, catalog, nil)
	testhelper.AssertDomainError(t, err, constant.CodeRolePermissionDuplicate)
	_, err = service.ResolvePermissions([]string{"tidak.ada"}, catalog, nil)
	testhelper.AssertDomainError(t, err, constant.CodeRolePermissionUnknown)
	_, err = service.ResolvePermissions([]string{"legacy.export"}, catalog, nil)
	testhelper.AssertDomainError(t, err, constant.CodeRolePermissionDeprecated)

	// Deprecated yang sudah terpasang boleh tetap.
	got, err = service.ResolvePermissions([]string{"legacy.export"}, catalog, map[string]bool{legacy.ID: true})
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestRole(t *testing.T) {
	tenant := "t1"
	r, err := entity.NewRole(entity.NewRoleParams{SolutionID: "s", TenantID: &tenant, Code: " HR_Admin ", Name: "HR Admin"}, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "hr_admin", r.Code.Value())
	assert.False(t, r.IsGlobal())
	assert.True(t, r.IsOwnedBy("t1"))
	assert.False(t, r.IsOwnedBy("t2"))
	assert.NoError(t, r.EnsureEditable())

	r.IsSystem = true
	testhelper.AssertDomainError(t, r.EnsureEditable(), constant.CodeRoleSystemImmutable)

	_, err = entity.NewRole(entity.NewRoleParams{SolutionID: "s", Code: "hr admin", Name: "X"}, time.Now())
	testhelper.AssertDomainError(t, err, constant.CodeRoleCodeInvalid)
}
