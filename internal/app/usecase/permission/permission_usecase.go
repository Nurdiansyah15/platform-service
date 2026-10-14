package permission

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/permission/constant"
	"platform-service/internal/domain/permission/entity"
	permrepo "platform-service/internal/domain/permission/repository"
	"platform-service/internal/domain/permission/service"
	"platform-service/internal/domain/permission/valueobject"
	solutionconstant "platform-service/internal/domain/solution/constant"
	solutionrepo "platform-service/internal/domain/solution/repository"
)

type PermissionDeps struct {
	Tx          port.Transactor
	Permissions permrepo.PermissionRepository
	Solutions   solutionrepo.SolutionRepository
	Query       port.PermissionQueryReadModel
}

type PermissionUseCase struct {
	deps PermissionDeps
	now  func() time.Time
}

func NewPermissionUseCase(deps PermissionDeps) *PermissionUseCase {
	return &PermissionUseCase{deps: deps, now: time.Now}
}

// Sync menyelaraskan katalog permission solution (spec §3.20). Idempotent: versi yang
// sama dengan versi tersimpan tidak mengubah apa pun. Permission yang hilang dari katalog
// menjadi deprecated. Baris solution dikunci selama sync agar deploy paralel tidak bentrok.
//
// Required — auth: service milik solution (scope catalog:sync) atau user platform admin | scope: catalog:sync
func (uc *PermissionUseCase) Sync(ctx context.Context, actor *principal.Principal, solutionID string, req dto.SyncPermissionsRequest) (*dto.SyncPermissionsResponse, error) {
	if !canSync(actor, solutionID) {
		return nil, mapPermissionDomainError(domainerr.New(constant.CodeCatalogSyncForbidden))
	}
	version, err := valueobject.NewCatalogVersion(req.Version)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	catalog := make([]entity.Definition, 0, len(req.Permissions))
	for _, p := range req.Permissions {
		d, err := entity.NewDefinition(p.Code, p.Name, p.Description)
		if err != nil {
			return nil, mapPermissionDomainError(err)
		}
		catalog = append(catalog, d)
	}

	resp := &dto.SyncPermissionsResponse{Version: version.Value()}
	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		sol, err := uc.deps.Solutions.FindByIDForUpdate(ctx, solutionID)
		if err != nil {
			return err
		}
		if sol.HasPermissionCatalogVersion(version.Value()) {
			return nil
		}
		existing, err := uc.deps.Permissions.ListBySolution(ctx, solutionID)
		if err != nil {
			return err
		}
		now := uc.now()
		plan, err := service.PlanCatalogSync(solutionID, existing, catalog, now)
		if err != nil {
			return err
		}
		for _, p := range plan.Changed {
			if err := uc.deps.Permissions.Save(ctx, p); err != nil {
				return err
			}
		}
		sol.SetPermissionCatalogVersion(version.Value(), now)
		if err := uc.deps.Solutions.Save(ctx, sol); err != nil {
			return err
		}
		resp.Changed = true
		resp.Created, resp.Updated, resp.Reactivated = plan.Created, plan.Updated, plan.Reactivated
		resp.Deprecated, resp.Unchanged = plan.Deprecated, plan.Unchanged
		return nil
	})
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	return resp, nil
}

// List menampilkan katalog permission solution. Solution inactive disembunyikan dari non-admin.
//
// Required — auth: user / service (token valid apa pun) | scope: -
func (uc *PermissionUseCase) List(ctx context.Context, actor *principal.Principal, solutionID string, req dto.ListPermissionQuery) ([]dto.PermissionResponse, dto.Meta, error) {
	sol, err := uc.deps.Solutions.FindByID(ctx, solutionID)
	if err != nil {
		return nil, dto.Meta{}, mapPermissionDomainError(err)
	}
	if !sol.IsActive() && !isPlatformAdmin(actor) {
		return nil, dto.Meta{}, mapPermissionDomainError(domainerr.New(solutionconstant.CodeSolutionNotFound))
	}
	items, meta, err := uc.deps.Query.List(ctx, port.PermissionListReadQuery{
		SolutionID: solutionID, Status: req.Status, Search: req.Search, PaginationParams: req.PaginationParams,
	})
	if err != nil {
		return nil, dto.Meta{}, mapPermissionDomainError(err)
	}
	return items, meta, nil
}
