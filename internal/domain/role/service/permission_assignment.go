// Package service berisi domain service role.
package service

import (
	"strings"

	domainerr "platform-service/internal/domain/errors"
	permentity "platform-service/internal/domain/permission/entity"
	permvo "platform-service/internal/domain/permission/valueobject"
	"platform-service/internal/domain/role/constant"
	"platform-service/internal/domain/role/entity"
)

// ResolvePermissions memetakan permission yang diminta ke permission solution untuk role
// (spec §2.6):
//   - code harus ada di katalog solution dan tidak dobel
//   - permission deprecated tidak boleh DITAMBAHKAN; yang sudah terpasang boleh tetap
//     (agar role lama tidak rusak)
//   - scope_type (kosong = company) harus company, branch, atau scope type custom solution;
//     scope type deprecated hanya boleh bila permission itu sudah terpasang dengan scope tsb
//
// catalog = seluruh permission solution; scopeTypes = scope type custom solution (semua
// status); current = permission ID → scope_type yang sudah terpasang.
func ResolvePermissions(in []entity.PermissionInput, catalog []*permentity.Permission, scopeTypes []*permentity.ScopeType,
	current map[string]string) ([]entity.PermissionGrant, error) {
	byCode := make(map[string]*permentity.Permission, len(catalog))
	for _, p := range catalog {
		byCode[p.Code.Value()] = p
	}
	scopeByCode := make(map[string]*permentity.ScopeType, len(scopeTypes))
	for _, s := range scopeTypes {
		scopeByCode[s.Code.Value()] = s
	}

	seen := make(map[string]bool, len(in))
	out := make([]entity.PermissionGrant, 0, len(in))
	for _, item := range in {
		code := strings.TrimSpace(item.Code)
		if seen[code] {
			return nil, domainerr.New(constant.CodeRolePermissionDuplicate)
		}
		seen[code] = true
		p, ok := byCode[code]
		if !ok {
			return nil, domainerr.New(constant.CodeRolePermissionUnknown)
		}
		prev, attached := current[p.ID]
		if !p.IsActive() && !attached {
			return nil, domainerr.New(constant.CodeRolePermissionDeprecated)
		}

		scope := strings.TrimSpace(item.ScopeType)
		if scope == "" {
			scope = entity.ScopeCompany
		}
		if !permvo.IsBuiltinScopeType(scope) {
			st, ok := scopeByCode[scope]
			keptDeprecated := attached && prev == scope
			if !ok || (!st.IsActive() && !keptDeprecated) {
				return nil, domainerr.New(constant.CodeRolePermissionScopeInvalid)
			}
		}
		out = append(out, entity.PermissionGrant{PermissionID: p.ID, Code: code, ScopeType: scope})
	}
	return out, nil
}
