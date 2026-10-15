// Package service berisi domain service otorisasi.
package service

import (
	"sort"

	"platform-service/internal/domain/authz/entity"
	"platform-service/internal/domain/authz/valueobject"
)

// Evaluate menentukan apakah grants memberi permission pada scope yang ditanyakan.
// Hierarki scope (keputusan Fase 15):
//   - assignment tenant  → berlaku di seluruh tenant (tenant, company, branch apa pun)
//   - assignment company → berlaku untuk company itu dan branch-branch di bawahnya
//   - assignment branch  → hanya branch itu
//
// Cek di level tenant hanya dipenuhi assignment level tenant. Mengembalikan code role
// (unik, terurut) yang memberi izin.
func Evaluate(grants []entity.Grant, permission string, check entity.CheckScope) (bool, []string) {
	roles := map[string]bool{}
	for _, g := range grants {
		if g.PermissionCode == permission && covers(g.Scope, check) {
			roles[g.RoleCode] = true
		}
	}
	if len(roles) == 0 {
		return false, nil
	}
	out := make([]string, 0, len(roles))
	for r := range roles {
		out = append(out, r)
	}
	sort.Strings(out)
	return true, out
}

func covers(granted valueobject.Scope, check entity.CheckScope) bool {
	switch granted.Type {
	case valueobject.ScopeTenant:
		return true
	case valueobject.ScopeCompany:
		switch check.Scope.Type {
		case valueobject.ScopeCompany:
			return check.Scope.ID == granted.ID
		case valueobject.ScopeBranch:
			return check.BranchCompanyID == granted.ID
		}
	case valueobject.ScopeBranch:
		return check.Scope.Type == valueobject.ScopeBranch && check.Scope.ID == granted.ID
	}
	return false
}
