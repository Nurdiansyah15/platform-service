package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/authz/constant"
	"platform-service/internal/domain/authz/entity"
	"platform-service/internal/domain/authz/service"
	vo "platform-service/internal/domain/authz/valueobject"
	orgservice "platform-service/internal/domain/organization/service"
	"platform-service/internal/testhelper"
)

type fakeGrants map[string][]entity.Grant // key: userID

func (f fakeGrants) Grants(_ context.Context, _, userID, solutionID string) ([]entity.Grant, error) {
	if solutionID != "console" {
		return nil, nil
	}
	return f[userID], nil
}

type fixedConsole string

func (c fixedConsole) ConsoleSolutionID(context.Context) (string, error) { return string(c), nil }

func TestCompanyAdminPolicy_EnsurePermission(t *testing.T) {
	grants := fakeGrants{
		"admin":  {grant("company_admin", constant.PermBranchManage, vo.ScopeCompany, "C1")},
		"kacab":  {grant("kepala_cabang", constant.PermBranchManage, vo.ScopeBranch, "B1")},
		"viewer": {grant("viewer", "report.read", vo.ScopeCompany, "C1")},
	}
	p := service.NewCompanyAdminPolicy(grants, fixedConsole("console"))

	tests := []struct {
		name  string
		a     orgservice.Accessor
		scope entity.CheckScope
		ok    bool
	}{
		{"platform admin", orgservice.Accessor{UserID: "x", IsPlatformAdmin: true}, service.CompanyScope("C1"), true},
		{"service token", orgservice.Accessor{ServiceSolutionID: "sol"}, service.CompanyScope("C1"), true},
		{"company admin at company", orgservice.Accessor{UserID: "admin"}, service.CompanyScope("C1"), true},
		{"company admin covers branch", orgservice.Accessor{UserID: "admin"}, service.BranchScope("C1", "B9"), true},
		{"company admin other company", orgservice.Accessor{UserID: "admin"}, service.CompanyScope("C2"), false},
		{"branch grant at own branch", orgservice.Accessor{UserID: "kacab"}, service.BranchScope("C1", "B1"), true},
		{"branch grant at other branch", orgservice.Accessor{UserID: "kacab"}, service.BranchScope("C1", "B2"), false},
		{"branch grant at company", orgservice.Accessor{UserID: "kacab"}, service.CompanyScope("C1"), false},
		{"no platform permission", orgservice.Accessor{UserID: "viewer"}, service.CompanyScope("C1"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := p.EnsurePermission(context.Background(), tc.a, "C1", constant.PermBranchManage, tc.scope)
			if tc.ok {
				require.NoError(t, err)
				return
			}
			testhelper.AssertDomainError(t, err, constant.CodeAdminPermissionDenied)
		})
	}
}

func TestCompanyAdminPolicy_EnsurePlatformRoleControl(t *testing.T) {
	all := func(scopeType vo.ScopeType, id string) []entity.Grant {
		var gs []entity.Grant
		for _, p := range constant.PlatformPermissions {
			gs = append(gs, grant("company_admin", p.Code, scopeType, id))
		}
		return gs
	}
	grants := fakeGrants{
		"admin":    all(vo.ScopeCompany, "C1"),
		"assigner": {grant("hr", constant.PermRoleAssign, vo.ScopeCompany, "C1")},
	}
	p := service.NewCompanyAdminPolicy(grants, fixedConsole("console"))
	ctx := context.Background()

	require.NoError(t, p.EnsurePlatformRoleControl(ctx, orgservice.Accessor{UserID: "admin"}, "C1", "console", service.CompanyScope("C1")))
	require.NoError(t, p.EnsurePlatformRoleControl(ctx, orgservice.Accessor{UserID: "assigner"}, "C1", "kasir", service.CompanyScope("C1")),
		"role solution lain tidak dibatasi")
	testhelper.AssertDomainError(t,
		p.EnsurePlatformRoleControl(ctx, orgservice.Accessor{UserID: "assigner"}, "C1", "console", service.CompanyScope("C1")),
		constant.CodeAdminPermissionDenied)
	require.NoError(t, p.EnsurePlatformRoleControl(ctx, orgservice.Accessor{ServiceSolutionID: "kasir"}, "C1", "console", service.CompanyScope("C1")))
}
