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

func in(pairs ...string) []entity.PermissionInput {
	out := []entity.PermissionInput{}
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, entity.PermissionInput{Code: pairs[i], ScopeType: pairs[i+1]})
	}
	return out
}

func scopeType(t *testing.T, code string, deprecated bool) *permentity.ScopeType {
	t.Helper()
	d, err := permentity.NewScopeTypeDefinition(code, code, nil, "", 0)
	require.NoError(t, err)
	st := permentity.NewScopeType("s", d, time.Now())
	if deprecated {
		st.Deprecate(time.Now())
	}
	return st
}

func TestResolvePermissions(t *testing.T) {
	read, write, legacy := perm(t, "employee.read", false), perm(t, "employee.write", false), perm(t, "legacy.export", true)
	catalog := []*permentity.Permission{read, write, legacy}
	scopes := []*permentity.ScopeType{scopeType(t, "self", false), scopeType(t, "old_unit", true)}

	got, err := service.ResolvePermissions(in("employee.read", "", "employee.write", "self"), catalog, scopes, nil)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "company", got[0].ScopeType, "kosong = company")
	assert.Equal(t, "self", got[1].ScopeType)
	assert.Equal(t, write.ID, got[1].PermissionID)

	_, err = service.ResolvePermissions(in("employee.read", "", "employee.read", "branch"), catalog, scopes, nil)
	testhelper.AssertDomainError(t, err, constant.CodeRolePermissionDuplicate)
	_, err = service.ResolvePermissions(in("tidak.ada", ""), catalog, scopes, nil)
	testhelper.AssertDomainError(t, err, constant.CodeRolePermissionUnknown)
	_, err = service.ResolvePermissions(in("legacy.export", ""), catalog, scopes, nil)
	testhelper.AssertDomainError(t, err, constant.CodeRolePermissionDeprecated)
	_, err = service.ResolvePermissions(in("employee.read", "region"), catalog, scopes, nil)
	testhelper.AssertDomainError(t, err, constant.CodeRolePermissionScopeInvalid)
	_, err = service.ResolvePermissions(in("employee.read", "old_unit"), catalog, scopes, nil)
	testhelper.AssertDomainError(t, err, constant.CodeRolePermissionScopeInvalid)

	// Yang sudah terpasang boleh tetap: permission deprecated & scope type deprecated (scope sama).
	got, err = service.ResolvePermissions(in("legacy.export", "", "employee.read", "old_unit"), catalog, scopes,
		map[string]string{legacy.ID: "company", read.ID: "old_unit"})
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestRole(t *testing.T) {
	companyID := "c1"
	r, err := entity.NewRole(entity.NewRoleParams{SolutionID: "s", CompanyID: &companyID, Code: " HR_Admin ", Name: "HR Admin"}, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "hr_admin", r.Code.Value())
	assert.False(t, r.IsGlobal())
	assert.True(t, r.IsOwnedBy("c1"))
	assert.False(t, r.IsOwnedBy("t2"))
	assert.NoError(t, r.EnsureEditable())

	r.IsSystem = true
	testhelper.AssertDomainError(t, r.EnsureEditable(), constant.CodeRoleSystemImmutable)

	_, err = entity.NewRole(entity.NewRoleParams{SolutionID: "s", Code: "hr admin", Name: "X"}, time.Now())
	testhelper.AssertDomainError(t, err, constant.CodeRoleCodeInvalid)
}
