// Package service berisi domain service role.
package service

import (
	domainerr "platform-service/internal/domain/errors"
	permentity "platform-service/internal/domain/permission/entity"
	"platform-service/internal/domain/role/constant"
)

// ResolvePermissions memetakan permission code ke permission solution untuk role (spec §2.6):
//   - code harus ada di katalog solution dan tidak dobel
//   - permission deprecated tidak boleh DITAMBAHKAN; yang sudah terpasang boleh tetap
//     (agar role lama tidak rusak)
//
// catalog = seluruh permission solution; current = ID permission yang sudah terpasang.
func ResolvePermissions(codes []string, catalog []*permentity.Permission, current map[string]bool) ([]*permentity.Permission, error) {
	byCode := make(map[string]*permentity.Permission, len(catalog))
	for _, p := range catalog {
		byCode[p.Code.Value()] = p
	}

	seen := make(map[string]bool, len(codes))
	out := make([]*permentity.Permission, 0, len(codes))
	for _, code := range codes {
		if seen[code] {
			return nil, domainerr.New(constant.CodeRolePermissionDuplicate)
		}
		seen[code] = true
		p, ok := byCode[code]
		if !ok {
			return nil, domainerr.New(constant.CodeRolePermissionUnknown)
		}
		if !p.IsActive() && !current[p.ID] {
			return nil, domainerr.New(constant.CodeRolePermissionDeprecated)
		}
		out = append(out, p)
	}
	return out, nil
}
