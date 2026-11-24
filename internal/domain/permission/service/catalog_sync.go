// Package service berisi domain service permission.
package service

import (
	"time"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/permission/constant"
	"platform-service/internal/domain/permission/entity"
)

// SyncPlan adalah hasil perencanaan sync satu jenis entri katalog (permission, group, scope type).
type SyncPlan[T any] struct {
	// Changed berisi entri yang harus disimpan (baru, berubah, diaktifkan ulang, di-deprecate).
	Changed     []T
	Created     int
	Updated     int
	Reactivated int
	Deprecated  int
	Unchanged   int
}

// catalogEntry — entri katalog tersimpan yang bisa diselaraskan dengan definisi D.
type catalogEntry[D any] interface {
	Apply(d D, now time.Time) (changed, reactivated bool)
	Deprecate(now time.Time) bool
}

// planSync membandingkan entri tersimpan dengan katalog baru (spec §3.20):
//   - code baru → dibuat active
//   - code yang ada → atribut diperbarui; deprecated → active lagi
//   - code yang tidak ada di katalog → deprecated (tidak dihapus)
func planSync[T catalogEntry[D], D any](
	existing []T, catalog []D, codeOf func(T) string, defCode func(D) string,
	create func(D) T, dup domainerr.Code, now time.Time,
) (*SyncPlan[T], error) {
	incoming := make(map[string]D, len(catalog))
	for _, d := range catalog {
		if _, ok := incoming[defCode(d)]; ok {
			return nil, domainerr.New(dup)
		}
		incoming[defCode(d)] = d
	}

	plan := &SyncPlan[T]{}
	seen := make(map[string]bool, len(existing))
	for _, e := range existing {
		seen[codeOf(e)] = true
		d, ok := incoming[codeOf(e)]
		if !ok {
			if e.Deprecate(now) {
				plan.Deprecated++
				plan.Changed = append(plan.Changed, e)
			} else {
				plan.Unchanged++
			}
			continue
		}
		changed, reactivated := e.Apply(d, now)
		switch {
		case reactivated:
			plan.Reactivated++
		case changed:
			plan.Updated++
		default:
			plan.Unchanged++
		}
		if changed {
			plan.Changed = append(plan.Changed, e)
		}
	}

	for _, d := range catalog {
		if seen[defCode(d)] {
			continue
		}
		plan.Created++
		plan.Changed = append(plan.Changed, create(d))
	}
	return plan, nil
}

// PlanCatalogSync merencanakan sync permission. Code ganda → CodePermissionCodeDuplicate.
// GroupID pada definisi harus sudah di-resolve oleh pemanggil. Hanya permission ber-source
// sync yang dikelola; permission manual tidak disentuh, dan katalog yang memuat code
// permission manual ditolak (CodePermissionSourceConflict).
func PlanCatalogSync(solutionID string, existing []*entity.Permission, catalog []entity.Definition, now time.Time) (*SyncPlan[*entity.Permission], error) {
	manual := map[string]bool{}
	synced := make([]*entity.Permission, 0, len(existing))
	for _, p := range existing {
		if p.IsManual() {
			manual[p.Code.Value()] = true
		} else {
			synced = append(synced, p)
		}
	}
	for _, d := range catalog {
		if manual[d.Code.Value()] {
			return nil, domainerr.New(constant.CodePermissionSourceConflict)
		}
	}
	return planSync(synced, catalog,
		func(p *entity.Permission) string { return p.Code.Value() },
		func(d entity.Definition) string { return d.Code.Value() },
		func(d entity.Definition) *entity.Permission { return entity.NewPermission(solutionID, d, now) },
		constant.CodePermissionCodeDuplicate, now)
}

// PlanGroupSync merencanakan sync permission group. Code ganda → CodeGroupDuplicate. Hanya
// group ber-source sync yang dikelola; katalog yang memuat code group manual ditolak
// (CodeGroupSourceConflict).
func PlanGroupSync(solutionID string, existing []*entity.Group, catalog []entity.GroupDefinition, now time.Time) (*SyncPlan[*entity.Group], error) {
	manual := map[string]bool{}
	synced := make([]*entity.Group, 0, len(existing))
	for _, g := range existing {
		if g.IsManual() {
			manual[g.Code.Value()] = true
		} else {
			synced = append(synced, g)
		}
	}
	for _, d := range catalog {
		if manual[d.Code.Value()] {
			return nil, domainerr.New(constant.CodeGroupSourceConflict)
		}
	}
	return planSync(synced, catalog,
		func(g *entity.Group) string { return g.Code.Value() },
		func(d entity.GroupDefinition) string { return d.Code.Value() },
		func(d entity.GroupDefinition) *entity.Group { return entity.NewGroup(solutionID, d, now) },
		constant.CodeGroupDuplicate, now)
}

// PlanScopeTypeSync merencanakan sync scope type custom. Code ganda → CodeScopeTypeDuplicate.
// Hanya scope type ber-source sync yang dikelola; katalog yang memuat code scope type manual
// ditolak (CodeScopeTypeSourceConflict).
func PlanScopeTypeSync(solutionID string, existing []*entity.ScopeType, catalog []entity.ScopeTypeDefinition, now time.Time) (*SyncPlan[*entity.ScopeType], error) {
	manual := map[string]bool{}
	synced := make([]*entity.ScopeType, 0, len(existing))
	for _, s := range existing {
		if s.IsManual() {
			manual[s.Code.Value()] = true
		} else {
			synced = append(synced, s)
		}
	}
	for _, d := range catalog {
		if manual[d.Code.Value()] {
			return nil, domainerr.New(constant.CodeScopeTypeSourceConflict)
		}
	}
	return planSync(synced, catalog,
		func(s *entity.ScopeType) string { return s.Code.Value() },
		func(d entity.ScopeTypeDefinition) string { return d.Code.Value() },
		func(d entity.ScopeTypeDefinition) *entity.ScopeType { return entity.NewScopeType(solutionID, d, now) },
		constant.CodeScopeTypeDuplicate, now)
}

// ValidateCatalogRefs memastikan setiap permission hanya merujuk group yang ada di katalog
// yang sama.
func ValidateCatalogRefs(perms []entity.Definition, groups []entity.GroupDefinition) error {
	groupSet := make(map[string]bool, len(groups))
	for _, g := range groups {
		groupSet[g.Code.Value()] = true
	}
	for _, p := range perms {
		if p.GroupCode != "" && !groupSet[p.GroupCode] {
			return domainerr.New(constant.CodeGroupUnknown)
		}
	}
	return nil
}

// ResolveManualRefs memvalidasi group permission manual terhadap group solution yang
// tersimpan dan mengisi GroupID. Group deprecated tetap boleh dirujuk (agar permission yang
// sudah merujuknya tetap bisa diedit).
func ResolveManualRefs(d entity.Definition, groups []*entity.Group) (entity.Definition, error) {
	d.GroupID = nil
	if d.GroupCode == "" {
		return d, nil
	}
	for _, g := range groups {
		if g.Code.Value() == d.GroupCode {
			id := g.ID
			d.GroupID = &id
			return d, nil
		}
	}
	return entity.Definition{}, domainerr.New(constant.CodeGroupUnknown)
}
