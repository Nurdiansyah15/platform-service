package permission

import (
	"context"
	"errors"
	"sort"
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
	Groups      permrepo.PermissionGroupRepository
	ScopeTypes  permrepo.ScopeTypeRepository
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

// Sync menyelaraskan katalog permission solution (spec §3.20) beserta scope type custom dan
// permission group (keduanya opsional). Idempotent: versi yang sama dengan versi tersimpan
// tidak mengubah apa pun. Entri yang hilang dari katalog menjadi deprecated. Baris solution
// dikunci selama sync agar deploy paralel tidak bentrok.
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
	scopeTypes := make([]entity.ScopeTypeDefinition, 0, len(req.ScopeTypes))
	for i, st := range req.ScopeTypes {
		d, err := entity.NewScopeTypeDefinition(st.Code, st.Name, st.Description, st.Parent, i)
		if err != nil {
			return nil, mapPermissionDomainError(err)
		}
		scopeTypes = append(scopeTypes, d)
	}
	groups := make([]entity.GroupDefinition, 0, len(req.Groups))
	for i, g := range req.Groups {
		d, err := entity.NewGroupDefinition(g.Code, g.Name, g.Description, i)
		if err != nil {
			return nil, mapPermissionDomainError(err)
		}
		if d, err = d.WithParent(g.Parent); err != nil {
			return nil, mapPermissionDomainError(err)
		}
		groups = append(groups, d)
	}
	catalog := make([]entity.Definition, 0, len(req.Permissions))
	for _, p := range req.Permissions {
		d, err := entity.NewDefinition(p.Code, p.Name, p.Description)
		if err != nil {
			return nil, mapPermissionDomainError(err)
		}
		if d, err = d.WithGroup(p.Group); err != nil {
			return nil, mapPermissionDomainError(err)
		}
		catalog = append(catalog, d)
	}
	if err := service.ValidateCatalogRefs(catalog, groups); err != nil {
		return nil, mapPermissionDomainError(err)
	}
	if err := service.ValidateGroupDefinitions(groups); err != nil {
		return nil, mapPermissionDomainError(err)
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
		now := uc.now()
		if resp.ScopeTypes, err = uc.syncScopeTypes(ctx, solutionID, scopeTypes, now); err != nil {
			return err
		}
		groupIDs, groupCounts, err := uc.syncGroups(ctx, solutionID, groups, now)
		if err != nil {
			return err
		}
		resp.Groups = groupCounts
		for i := range catalog {
			if catalog[i].GroupCode != "" {
				id := groupIDs[catalog[i].GroupCode]
				catalog[i].GroupID = &id
			}
		}

		existing, err := uc.deps.Permissions.ListBySolution(ctx, solutionID)
		if err != nil {
			return err
		}
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
	if err := uc.ensureSolutionVisible(ctx, actor, solutionID); err != nil {
		return nil, dto.Meta{}, mapPermissionDomainError(err)
	}
	items, meta, err := uc.deps.Query.List(ctx, port.PermissionListReadQuery{
		SolutionID: solutionID, Status: req.Status, Search: req.Search, Group: req.Group, Source: req.Source,
		PaginationParams: req.PaginationParams,
	})
	if err != nil {
		return nil, dto.Meta{}, mapPermissionDomainError(err)
	}
	return items, meta, nil
}

func (uc *PermissionUseCase) syncScopeTypes(ctx context.Context, solutionID string, catalog []entity.ScopeTypeDefinition, now time.Time) (dto.SyncCounts, error) {
	existing, err := uc.deps.ScopeTypes.ListBySolution(ctx, solutionID)
	if err != nil {
		return dto.SyncCounts{}, err
	}
	plan, err := service.PlanScopeTypeSync(solutionID, existing, catalog, now)
	if err != nil {
		return dto.SyncCounts{}, err
	}
	for _, st := range plan.Changed {
		if err := uc.deps.ScopeTypes.Save(ctx, st); err != nil {
			return dto.SyncCounts{}, err
		}
	}
	return countsOf(plan), nil
}

// syncGroups menyimpan group dan mengembalikan peta code → id (semua group solution).
func (uc *PermissionUseCase) syncGroups(ctx context.Context, solutionID string, catalog []entity.GroupDefinition, now time.Time) (map[string]string, dto.SyncCounts, error) {
	existing, err := uc.deps.Groups.ListBySolution(ctx, solutionID)
	if err != nil {
		return nil, dto.SyncCounts{}, err
	}
	plan, err := service.PlanGroupSync(solutionID, existing, catalog, now)
	if err != nil {
		return nil, dto.SyncCounts{}, err
	}
	ids := make(map[string]string, len(existing)+plan.Created)
	all := append([]*entity.Group{}, existing...)
	for _, g := range existing {
		ids[g.Code.Value()] = g.ID
	}
	for _, g := range plan.Changed {
		if _, ok := ids[g.Code.Value()]; !ok {
			all = append(all, g)
		}
	}
	// Induk disimpan sebelum anaknya (FK parent_id).
	depth := func(g *entity.Group) int { return len(service.GroupPath(all, g.ID)) }
	sort.SliceStable(plan.Changed, func(i, j int) bool { return depth(plan.Changed[i]) < depth(plan.Changed[j]) })
	for _, g := range plan.Changed {
		if err := uc.deps.Groups.Save(ctx, g); err != nil {
			return nil, dto.SyncCounts{}, err
		}
		ids[g.Code.Value()] = g.ID
	}
	return ids, countsOf(plan), nil
}

func countsOf[T any](p *service.SyncPlan[T]) dto.SyncCounts {
	return dto.SyncCounts{
		Created: p.Created, Updated: p.Updated, Reactivated: p.Reactivated, Deprecated: p.Deprecated, Unchanged: p.Unchanged,
	}
}

// ListGroups menampilkan permission group solution (urut tampil).
//
// Required — auth: user / service (token valid apa pun) | scope: -
func (uc *PermissionUseCase) ListGroups(ctx context.Context, actor *principal.Principal, solutionID string, req dto.ListCatalogQuery) ([]dto.PermissionGroupResponse, error) {
	if err := uc.ensureSolutionVisible(ctx, actor, solutionID); err != nil {
		return nil, mapPermissionDomainError(err)
	}
	groups, err := uc.deps.Groups.ListBySolution(ctx, solutionID)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	out := make([]dto.PermissionGroupResponse, 0, len(groups))
	for _, g := range groups {
		if req.Status != nil && g.Status.String() != *req.Status {
			continue
		}
		out = append(out, *toGroupResponse(g, groups))
	}
	return out, nil
}

// ListScopeTypes menampilkan scope type yang bisa dipakai assignment role solution:
// company & branch (built-in) di depan, lalu scope type custom (urut tampil).
//
// Required — auth: user / service (token valid apa pun) | scope: -
func (uc *PermissionUseCase) ListScopeTypes(ctx context.Context, actor *principal.Principal, solutionID string, req dto.ListCatalogQuery) ([]dto.ScopeTypeResponse, error) {
	if err := uc.ensureSolutionVisible(ctx, actor, solutionID); err != nil {
		return nil, mapPermissionDomainError(err)
	}
	custom, err := uc.deps.ScopeTypes.ListBySolution(ctx, solutionID)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	active := valueobject.StatusActive.String()
	out := make([]dto.ScopeTypeResponse, 0, len(custom)+2)
	if req.Status == nil || *req.Status == active {
		out = append(out,
			dto.ScopeTypeResponse{Code: constant.ScopeTypeCompany, Name: "Company", Builtin: true, Status: active, Source: sourceBuiltin},
			dto.ScopeTypeResponse{Code: constant.ScopeTypeBranch, Name: "Branch", Parent: constant.ScopeTypeCompany, Builtin: true,
				Status: active, Source: sourceBuiltin},
		)
	}
	for _, st := range custom {
		if req.Status != nil && st.Status.String() != *req.Status {
			continue
		}
		out = append(out, *toScopeTypeResponse(st))
	}
	return out, nil
}

// ensureSolutionVisible — solution harus ada; solution inactive disembunyikan dari non-admin.
func (uc *PermissionUseCase) ensureSolutionVisible(ctx context.Context, actor *principal.Principal, solutionID string) error {
	sol, err := uc.deps.Solutions.FindByID(ctx, solutionID)
	if err != nil {
		return err
	}
	if !sol.IsActive() && !isPlatformAdmin(actor) {
		return domainerr.New(solutionconstant.CodeSolutionNotFound)
	}
	return nil
}

// Create membuat permission manual. Code yang sudah dipakai (manual atau hasil sync) → 409.
// Permission manual tidak pernah di-deprecate oleh sync; sync yang memuat code-nya ditolak.
//
// Required — auth: user (platform admin) | scope: -
func (uc *PermissionUseCase) Create(ctx context.Context, solutionID string, req dto.CreatePermissionRequest) (*dto.PermissionResponse, error) {
	if _, err := uc.deps.Solutions.FindByID(ctx, solutionID); err != nil {
		return nil, mapPermissionDomainError(err)
	}
	d, err := entity.NewDefinition(req.Code, req.Name, req.Description)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	if d, err = d.WithGroup(req.Group); err != nil {
		return nil, mapPermissionDomainError(err)
	}
	groups, err := uc.deps.Groups.ListBySolution(ctx, solutionID)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	if d, err = service.ResolveManualRefs(d, groups); err != nil {
		return nil, mapPermissionDomainError(err)
	}
	p := entity.NewManualPermission(solutionID, d, uc.now())
	if err := uc.deps.Permissions.Save(ctx, p); err != nil {
		var de *domainerr.DomainError
		if errors.As(err, &de) && de.Code == constant.CodePermissionCodeDuplicate {
			err = domainerr.New(constant.CodePermissionCodeTaken)
		}
		return nil, mapPermissionDomainError(err)
	}
	return toPermissionResponse(p, groups), nil
}

// Get menampilkan satu permission. Solution inactive disembunyikan dari non-admin.
//
// Required — auth: user / service (token valid apa pun) | scope: -
func (uc *PermissionUseCase) Get(ctx context.Context, actor *principal.Principal, permissionID string) (*dto.PermissionResponse, error) {
	p, err := uc.deps.Permissions.FindByID(ctx, permissionID)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	if err := uc.ensureSolutionVisible(ctx, actor, p.SolutionID); err != nil {
		if isNotFound(err) {
			err = domainerr.New(constant.CodePermissionNotFound)
		}
		return nil, mapPermissionDomainError(err)
	}
	groups, err := uc.deps.Groups.ListBySolution(ctx, p.SolutionID)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	return toPermissionResponse(p, groups), nil
}

// Update mengubah permission manual (nama, deskripsi, grup, status). Permission hasil
// sync read-only (409) karena dikelola kode solution.
//
// Required — auth: user (platform admin) | scope: -
func (uc *PermissionUseCase) Update(ctx context.Context, permissionID string, req dto.UpdatePermissionRequest) (*dto.PermissionResponse, error) {
	p, err := uc.deps.Permissions.FindByID(ctx, permissionID)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	if err := p.EnsureManual(); err != nil {
		return nil, mapPermissionDomainError(err)
	}
	groups, err := uc.deps.Groups.ListBySolution(ctx, p.SolutionID)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}

	name, description := p.Name, p.Description
	if req.Name != nil {
		name = *req.Name
	}
	if req.Description != nil {
		description = req.Description
	}
	d, err := entity.NewDefinition(p.Code.Value(), name, description)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	group := currentGroupCode(p, groups)
	if req.Group != nil {
		group = req.Group
	}
	if d, err = d.WithGroup(group); err != nil {
		return nil, mapPermissionDomainError(err)
	}
	if d, err = service.ResolveManualRefs(d, groups); err != nil {
		return nil, mapPermissionDomainError(err)
	}

	now := uc.now()
	changed := p.Edit(d, now)
	if req.Status != nil {
		st, err := valueobject.NewPermissionStatus(*req.Status)
		if err != nil {
			return nil, mapPermissionDomainError(err)
		}
		changed = p.SetStatus(st, now) || changed
	}
	if changed {
		if err := uc.deps.Permissions.Save(ctx, p); err != nil {
			return nil, mapPermissionDomainError(err)
		}
	}
	return toPermissionResponse(p, groups), nil
}

// CreateGroup membuat permission group manual. Code yang sudah dipakai (manual atau hasil
// sync) → 409. Group manual tidak disentuh sync; sync yang memuat code-nya ditolak.
//
// Required — auth: user (platform admin) | scope: -
func (uc *PermissionUseCase) CreateGroup(ctx context.Context, solutionID string, req dto.CreatePermissionGroupRequest) (*dto.PermissionGroupResponse, error) {
	if _, err := uc.deps.Solutions.FindByID(ctx, solutionID); err != nil {
		return nil, mapPermissionDomainError(err)
	}
	existing, err := uc.deps.Groups.ListBySolution(ctx, solutionID)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	order := 0
	for _, g := range existing {
		if g.SortOrder >= order {
			order = g.SortOrder + 1
		}
	}
	if req.SortOrder != nil {
		order = *req.SortOrder
	}
	d, err := entity.NewGroupDefinition(req.Code, req.Name, req.Description, order)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	if d, err = d.WithParent(req.Parent); err != nil {
		return nil, mapPermissionDomainError(err)
	}
	now := uc.now()
	g := entity.NewManualGroup(solutionID, d, now)
	if _, err := service.PlaceGroup(existing, g, d.ParentCode, now); err != nil {
		return nil, mapPermissionDomainError(err)
	}
	if err := uc.deps.Groups.Save(ctx, g); err != nil {
		var de *domainerr.DomainError
		if errors.As(err, &de) && de.Code == constant.CodeGroupDuplicate {
			err = domainerr.New(constant.CodeGroupCodeTaken)
		}
		return nil, mapPermissionDomainError(err)
	}
	return toGroupResponse(g, append(existing, g)), nil
}

// GetGroup menampilkan satu permission group. Solution inactive disembunyikan dari non-admin.
//
// Required — auth: user / service (token valid apa pun) | scope: -
func (uc *PermissionUseCase) GetGroup(ctx context.Context, actor *principal.Principal, groupID string) (*dto.PermissionGroupResponse, error) {
	g, err := uc.deps.Groups.FindByID(ctx, groupID)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	if err := uc.ensureSolutionVisible(ctx, actor, g.SolutionID); err != nil {
		if isNotFound(err) {
			err = domainerr.New(constant.CodeGroupNotFound)
		}
		return nil, mapPermissionDomainError(err)
	}
	groups, err := uc.deps.Groups.ListBySolution(ctx, g.SolutionID)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	return toGroupResponse(g, groups), nil
}

// UpdateGroup mengubah group manual (nama, deskripsi, urutan, status). Group hasil sync
// read-only (409). Group deprecated tidak bisa dipilih untuk permission baru; permission yang
// sudah memakainya tetap tampil di grup itu.
//
// Required — auth: user (platform admin) | scope: -
func (uc *PermissionUseCase) UpdateGroup(ctx context.Context, groupID string, req dto.UpdatePermissionGroupRequest) (*dto.PermissionGroupResponse, error) {
	g, err := uc.deps.Groups.FindByID(ctx, groupID)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	if err := g.EnsureManual(); err != nil {
		return nil, mapPermissionDomainError(err)
	}
	name, description, order := g.Name, g.Description, g.SortOrder
	if req.Name != nil {
		name = *req.Name
	}
	if req.Description != nil {
		description = req.Description
	}
	if req.SortOrder != nil {
		order = *req.SortOrder
	}
	d, err := entity.NewGroupDefinition(g.Code.Value(), name, description, order)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	groups, err := uc.deps.Groups.ListBySolution(ctx, g.SolutionID)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	now := uc.now()
	changed := false
	if req.Parent != nil {
		if d, err = d.WithParent(req.Parent); err != nil {
			return nil, mapPermissionDomainError(err)
		}
		if changed, err = service.PlaceGroup(groups, g, d.ParentCode, now); err != nil {
			return nil, mapPermissionDomainError(err)
		}
	}
	changed = g.Edit(d, now) || changed
	if req.Status != nil {
		st, err := valueobject.NewPermissionStatus(*req.Status)
		if err != nil {
			return nil, mapPermissionDomainError(err)
		}
		changed = g.SetStatus(st, now) || changed
	}
	if changed {
		if err := uc.deps.Groups.Save(ctx, g); err != nil {
			return nil, mapPermissionDomainError(err)
		}
	}
	return toGroupResponse(g, groups), nil
}

// CreateScopeType membuat scope type custom manual. Code yang sudah dipakai (manual atau
// hasil sync) → 409. Scope type manual tidak disentuh sync; sync yang memuat code-nya ditolak.
//
// Required — auth: user (platform admin) | scope: -
func (uc *PermissionUseCase) CreateScopeType(ctx context.Context, solutionID string, req dto.CreateScopeTypeRequest) (*dto.ScopeTypeResponse, error) {
	if _, err := uc.deps.Solutions.FindByID(ctx, solutionID); err != nil {
		return nil, mapPermissionDomainError(err)
	}
	existing, err := uc.deps.ScopeTypes.ListBySolution(ctx, solutionID)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	order := 0
	for _, st := range existing {
		if st.SortOrder >= order {
			order = st.SortOrder + 1
		}
	}
	if req.SortOrder != nil {
		order = *req.SortOrder
	}
	d, err := entity.NewScopeTypeDefinition(req.Code, req.Name, req.Description, req.Parent, order)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	st := entity.NewManualScopeType(solutionID, d, uc.now())
	if err := uc.deps.ScopeTypes.Save(ctx, st); err != nil {
		var de *domainerr.DomainError
		if errors.As(err, &de) && de.Code == constant.CodeScopeTypeDuplicate {
			err = domainerr.New(constant.CodeScopeTypeCodeTaken)
		}
		return nil, mapPermissionDomainError(err)
	}
	return toScopeTypeResponse(st), nil
}

// GetScopeType menampilkan satu scope type custom. Solution inactive disembunyikan dari non-admin.
//
// Required — auth: user / service (token valid apa pun) | scope: -
func (uc *PermissionUseCase) GetScopeType(ctx context.Context, actor *principal.Principal, scopeTypeID string) (*dto.ScopeTypeResponse, error) {
	st, err := uc.deps.ScopeTypes.FindByID(ctx, scopeTypeID)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	if err := uc.ensureSolutionVisible(ctx, actor, st.SolutionID); err != nil {
		if isNotFound(err) {
			err = domainerr.New(constant.CodeScopeTypeNotFound)
		}
		return nil, mapPermissionDomainError(err)
	}
	return toScopeTypeResponse(st), nil
}

// UpdateScopeType mengubah scope type manual (nama, deskripsi, parent, urutan, status). Scope
// type hasil sync read-only (409). Deprecated = tidak bisa dipakai assignment baru;
// assignment lama tetap dievaluasi.
//
// Required — auth: user (platform admin) | scope: -
func (uc *PermissionUseCase) UpdateScopeType(ctx context.Context, scopeTypeID string, req dto.UpdateScopeTypeRequest) (*dto.ScopeTypeResponse, error) {
	st, err := uc.deps.ScopeTypes.FindByID(ctx, scopeTypeID)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	if err := st.EnsureManual(); err != nil {
		return nil, mapPermissionDomainError(err)
	}
	name, description, parent, order := st.Name, st.Description, st.Parent.String(), st.SortOrder
	if req.Name != nil {
		name = *req.Name
	}
	if req.Description != nil {
		description = req.Description
	}
	if req.Parent != nil {
		parent = *req.Parent
	}
	if req.SortOrder != nil {
		order = *req.SortOrder
	}
	d, err := entity.NewScopeTypeDefinition(st.Code.Value(), name, description, parent, order)
	if err != nil {
		return nil, mapPermissionDomainError(err)
	}
	now := uc.now()
	changed := st.Edit(d, now)
	if req.Status != nil {
		status, err := valueobject.NewPermissionStatus(*req.Status)
		if err != nil {
			return nil, mapPermissionDomainError(err)
		}
		changed = st.SetStatus(status, now) || changed
	}
	if changed {
		if err := uc.deps.ScopeTypes.Save(ctx, st); err != nil {
			return nil, mapPermissionDomainError(err)
		}
	}
	return toScopeTypeResponse(st), nil
}
