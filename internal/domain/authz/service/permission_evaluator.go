package service

import (
	"sort"

	"platform-service/internal/domain/authz/entity"
	"platform-service/internal/domain/authz/valueobject"
)

// Evaluate menentukan apakah grants memberi permission pada scope yang ditanyakan.
// Hierarki scope (keputusan Fase 15; company = entitas tertinggi):
//   - grant company → berlaku untuk company itu dan semua scope di bawahnya (branch & custom)
//   - grant branch  → branch itu, plus scope custom ber-parent branch yang menyebut branch itu
//   - grant custom  → hanya scope custom yang sama persis
//
// Scope grant ditentukan role (role_permissions.scope_type) + nilai assignment.
// Cek di level company hanya dipenuhi assignment level company. Mengembalikan code
// role (unik, terurut) yang memberi izin.
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
	case valueobject.ScopeCompany:
		return check.CompanyID == granted.ID
	case valueobject.ScopeBranch:
		if check.Scope.Type == valueobject.ScopeBranch {
			return check.Scope.ID == granted.ID
		}
		return !check.Scope.Type.IsBuiltin() && check.ParentBranchID == granted.ID
	default:
		return check.Scope.Type == granted.Type && check.Scope.ID == granted.ID
	}
}
