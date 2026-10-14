// Package service berisi domain service permission.
package service

import (
	"time"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/permission/constant"
	"platform-service/internal/domain/permission/entity"
)

// SyncPlan adalah hasil perencanaan sync katalog permission.
type SyncPlan struct {
	// Changed berisi permission yang harus disimpan (baru, berubah, diaktifkan ulang, di-deprecate).
	Changed     []*entity.Permission
	Created     int
	Updated     int
	Reactivated int
	Deprecated  int
	Unchanged   int
}

// PlanCatalogSync membandingkan permission yang ada dengan katalog baru (spec §3.20):
//   - code baru → dibuat active
//   - code yang ada → nama/deskripsi diperbarui; deprecated → active lagi
//   - code yang tidak ada di katalog → deprecated (tidak dihapus)
//
// Code ganda di katalog → CodePermissionCodeDuplicate.
func PlanCatalogSync(solutionID string, existing []*entity.Permission, catalog []entity.Definition, now time.Time) (*SyncPlan, error) {
	incoming := make(map[string]entity.Definition, len(catalog))
	for _, d := range catalog {
		if _, dup := incoming[d.Code.Value()]; dup {
			return nil, domainerr.New(constant.CodePermissionCodeDuplicate)
		}
		incoming[d.Code.Value()] = d
	}

	plan := &SyncPlan{}
	seen := make(map[string]bool, len(existing))
	for _, p := range existing {
		seen[p.Code.Value()] = true
		d, ok := incoming[p.Code.Value()]
		if !ok {
			if p.Deprecate(now) {
				plan.Deprecated++
				plan.Changed = append(plan.Changed, p)
			} else {
				plan.Unchanged++
			}
			continue
		}
		changed, reactivated := p.Apply(d, now)
		switch {
		case reactivated:
			plan.Reactivated++
		case changed:
			plan.Updated++
		default:
			plan.Unchanged++
		}
		if changed {
			plan.Changed = append(plan.Changed, p)
		}
	}

	for _, d := range catalog {
		if seen[d.Code.Value()] {
			continue
		}
		plan.Created++
		plan.Changed = append(plan.Changed, entity.NewPermission(solutionID, d, now))
	}
	return plan, nil
}
