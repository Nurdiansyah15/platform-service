package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"platform-service/internal/domain/authz/entity"
	"platform-service/internal/domain/authz/service"
	vo "platform-service/internal/domain/authz/valueobject"
)

func grant(role, perm string, t vo.ScopeType, id string) entity.Grant {
	return entity.Grant{RoleCode: role, PermissionCode: perm, Scope: vo.Scope{Type: t, ID: id}}
}

func check(t vo.ScopeType, id, company string) entity.CheckScope {
	if company == "" {
		company = id
	}
	return entity.CheckScope{Scope: vo.Scope{Type: t, ID: id}, CompanyID: company}
}

func customCheck(t vo.ScopeType, id, company, parentBranch string) entity.CheckScope {
	return entity.CheckScope{Scope: vo.Scope{Type: t, ID: id}, CompanyID: company, ParentBranchID: parentBranch}
}

func TestEvaluate(t *testing.T) {
	grants := []entity.Grant{
		grant("hr_admin", "employee.write", vo.ScopeCompany, "C1"),
		grant("manager", "leave.approve", vo.ScopeCompany, "C1"),
		grant("supervisor", "attendance.edit", vo.ScopeBranch, "B1"),
		grant("viewer", "employee.read", vo.ScopeCompany, "C1"),
		grant("hr_admin", "employee.read", vo.ScopeCompany, "C1"),
	}

	tests := []struct {
		name       string
		permission string
		check      entity.CheckScope
		allowed    bool
		roles      []string
	}{
		{"company grant, cek company", "employee.write", check(vo.ScopeCompany, "C1", ""), true, []string{"hr_admin"}},
		{"company grant berlaku di branch", "employee.write", check(vo.ScopeBranch, "B9", "C1"), true, []string{"hr_admin"}},
		{"company grant, company sama", "leave.approve", check(vo.ScopeCompany, "C1", ""), true, []string{"manager"}},
		{"company grant, branch di bawahnya", "leave.approve", check(vo.ScopeBranch, "B1", "C1"), true, []string{"manager"}},
		{"company grant, company lain", "leave.approve", check(vo.ScopeCompany, "C2", ""), false, nil},
		{"branch grant, branch sama", "attendance.edit", check(vo.ScopeBranch, "B1", "C1"), true, []string{"supervisor"}},
		{"branch grant, branch lain", "attendance.edit", check(vo.ScopeBranch, "B2", "C1"), false, nil},
		{"branch grant, cek company induk", "attendance.edit", check(vo.ScopeCompany, "C1", ""), false, nil},
		{"beberapa role memberi izin", "employee.read", check(vo.ScopeCompany, "C1", ""), true, []string{"hr_admin", "viewer"}},
		{"permission tidak dimiliki", "payroll.run", check(vo.ScopeCompany, "C1", ""), false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, roles := service.Evaluate(grants, tt.permission, tt.check)
			assert.Equal(t, tt.allowed, allowed)
			assert.Equal(t, tt.roles, roles)
		})
	}
}

func TestEvaluate_CustomScope(t *testing.T) {
	const dept vo.ScopeType = "department"
	const project vo.ScopeType = "project"
	grants := []entity.Grant{
		grant("hr_admin", "leave.approve", vo.ScopeCompany, "C1"),
		grant("kacab", "leave.view", vo.ScopeBranch, "B1"),
		grant("kadep", "leave.edit", dept, "D1"),
	}

	tests := []struct {
		name       string
		permission string
		check      entity.CheckScope
		allowed    bool
		roles      []string
	}{
		{"company grant menutupi scope custom", "leave.approve", customCheck(dept, "D9", "C1", ""), true, []string{"hr_admin"}},
		{"company grant, company lain", "leave.approve", customCheck(dept, "D9", "C2", ""), false, nil},
		{"branch grant, custom ber-parent branch itu", "leave.view", customCheck(dept, "D1", "C1", "B1"), true, []string{"kacab"}},
		{"branch grant, custom tanpa parent", "leave.view", customCheck(dept, "D1", "C1", ""), false, nil},
		{"branch grant, custom di branch lain", "leave.view", customCheck(dept, "D1", "C1", "B2"), false, nil},
		{"custom grant, scope sama", "leave.edit", customCheck(dept, "D1", "C1", ""), true, []string{"kadep"}},
		{"custom grant, id lain", "leave.edit", customCheck(dept, "D2", "C1", ""), false, nil},
		{"custom grant, type lain id sama", "leave.edit", customCheck(project, "D1", "C1", ""), false, nil},
		{"custom grant tidak menutupi company", "leave.edit", check(vo.ScopeCompany, "C1", ""), false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, roles := service.Evaluate(grants, tt.permission, tt.check)
			assert.Equal(t, tt.allowed, allowed)
			assert.Equal(t, tt.roles, roles)
		})
	}
}
