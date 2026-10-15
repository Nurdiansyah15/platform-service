package role

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	domainerr "platform-service/internal/domain/errors"
	permrepo "platform-service/internal/domain/permission/repository"
	"platform-service/internal/domain/role/constant"
	"platform-service/internal/domain/role/entity"
	"platform-service/internal/domain/role/event"
	rolerepo "platform-service/internal/domain/role/repository"
	roleservice "platform-service/internal/domain/role/service"
	solutionconstant "platform-service/internal/domain/solution/constant"
	solutionrepo "platform-service/internal/domain/solution/repository"
	subrepo "platform-service/internal/domain/subscription/repository"
	tenantconstant "platform-service/internal/domain/tenant/constant"
	tenantrepo "platform-service/internal/domain/tenant/repository"
	tenantservice "platform-service/internal/domain/tenant/service"
)

type RoleDeps struct {
	Tx              port.Transactor
	Outbox          port.EventOutbox
	Roles           rolerepo.RoleRepository
	RolePermissions rolerepo.RolePermissionRepository
	Permissions     permrepo.PermissionRepository
	Solutions       solutionrepo.SolutionRepository
	Tenants         tenantrepo.TenantRepository
	Members         tenantrepo.TenantMemberRepository
	Subscriptions   subrepo.SubscriptionRepository
	Query           port.RoleQueryReadModel
}

type RoleUseCase struct {
	deps   RoleDeps
	access *tenantservice.AccessPolicy
	now    func() time.Time
}

func NewRoleUseCase(deps RoleDeps) *RoleUseCase {
	return &RoleUseCase{deps: deps, access: tenantservice.NewAccessPolicy(deps.Members, deps.Subscriptions), now: time.Now}
}

// ── Role global (platform) ───────────────────────────────────────────────────

// ListGlobal menampilkan role global solution (tenant_id NULL).
//
// Required — auth: user / service (token valid apa pun) | scope: -
func (uc *RoleUseCase) ListGlobal(ctx context.Context, actor *principal.Principal, solutionID string, req dto.ListRoleQuery) ([]dto.RoleResponse, dto.Meta, error) {
	if err := uc.ensureSolutionVisible(ctx, actor, solutionID); err != nil {
		return nil, dto.Meta{}, err
	}
	return uc.list(ctx, port.RoleListReadQuery{SolutionID: solutionID, Search: req.Search, PaginationParams: req.PaginationParams})
}

// CreateGlobal membuat role global (template) untuk solution.
//
// Required — auth: user (platform admin) | scope: -
func (uc *RoleUseCase) CreateGlobal(ctx context.Context, solutionID string, req dto.CreateRoleRequest) (*dto.RoleResponse, error) {
	if _, err := uc.deps.Solutions.FindByID(ctx, solutionID); err != nil {
		return nil, mapRoleDomainError(err)
	}
	return uc.create(ctx, entity.NewRoleParams{SolutionID: solutionID, Code: req.Code, Name: req.Name, Description: req.Description})
}

// ListPermissions menampilkan permission role. Custom role hanya untuk tenant pemilik
// (selain itu 404).
//
// Required — auth: user / service | scope: -
func (uc *RoleUseCase) ListPermissions(ctx context.Context, actor *principal.Principal, roleID string) ([]dto.RolePermissionResponse, error) {
	r, err := uc.deps.Roles.FindByID(ctx, roleID)
	if err != nil {
		return nil, mapRoleDomainError(err)
	}
	if r.IsGlobal() {
		if err := uc.ensureSolutionVisible(ctx, actor, r.SolutionID); err != nil {
			return nil, mapRoleDomainError(domainerr.New(constant.CodeRoleNotFound))
		}
	} else if err := uc.ensureTenantRoleAccess(ctx, actor, *r.TenantID, r); err != nil {
		return nil, err
	}
	return uc.permissions(ctx, r.ID)
}

// ReplaceGlobalPermissions mengganti seluruh permission role global. Role sistem tidak
// bisa diubah; custom role tenant harus lewat endpoint tenant (404 di sini).
//
// Required — auth: user (platform admin) | scope: -
func (uc *RoleUseCase) ReplaceGlobalPermissions(ctx context.Context, roleID string, req dto.ReplaceRolePermissionsRequest) ([]dto.RolePermissionResponse, error) {
	r, err := uc.deps.Roles.FindByID(ctx, roleID)
	if err != nil {
		return nil, mapRoleDomainError(err)
	}
	if !r.IsGlobal() {
		return nil, mapRoleDomainError(domainerr.New(constant.CodeRoleNotFound))
	}
	return uc.replacePermissions(ctx, r, req.PermissionCodes)
}

// ── Role custom tenant ───────────────────────────────────────────────────────

// ListForTenant menampilkan role global + custom role milik tenant untuk solution.
//
// Required — auth: user (platform admin / member dengan tid) atau service milik solution | scope: roles:write
func (uc *RoleUseCase) ListForTenant(ctx context.Context, actor *principal.Principal, tenantID, solutionID string, req dto.ListRoleQuery) ([]dto.RoleResponse, dto.Meta, error) {
	if err := uc.ensureTenantSolutionAccess(ctx, actor, tenantID, solutionID); err != nil {
		return nil, dto.Meta{}, err
	}
	return uc.list(ctx, port.RoleListReadQuery{SolutionID: solutionID, TenantID: &tenantID, Search: req.Search, PaginationParams: req.PaginationParams})
}

// CreateForTenant membuat custom role milik tenant. Tenant wajib subscribe solution
// (trialing / active).
//
// Required — auth: user (platform admin / member dengan tid) atau service milik solution | scope: roles:write
func (uc *RoleUseCase) CreateForTenant(ctx context.Context, actor *principal.Principal, tenantID, solutionID string, req dto.CreateRoleRequest) (*dto.RoleResponse, error) {
	if err := uc.ensureTenantSolutionAccess(ctx, actor, tenantID, solutionID); err != nil {
		return nil, err
	}
	ok, err := uc.deps.Subscriptions.HasUsable(ctx, tenantID, solutionID)
	if err != nil {
		return nil, mapRoleDomainError(err)
	}
	if !ok {
		return nil, mapRoleDomainError(domainerr.New(constant.CodeRoleTenantNotSubscribed))
	}
	return uc.create(ctx, entity.NewRoleParams{SolutionID: solutionID, TenantID: &tenantID, Code: req.Code, Name: req.Name, Description: req.Description})
}

// ReplaceTenantRolePermissions mengganti permission custom role milik tenant. Role global
// atau milik tenant lain → 404.
//
// Required — auth: user (platform admin / member dengan tid) atau service milik solution | scope: roles:write
func (uc *RoleUseCase) ReplaceTenantRolePermissions(ctx context.Context, actor *principal.Principal, tenantID, roleID string, req dto.ReplaceRolePermissionsRequest) ([]dto.RolePermissionResponse, error) {
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), tenantID); err != nil {
		return nil, mapRoleDomainError(err)
	}
	r, err := uc.deps.Roles.FindByID(ctx, roleID)
	if err != nil {
		return nil, mapRoleDomainError(err)
	}
	if !r.IsOwnedBy(tenantID) || (actor.IsService() && actor.SolutionID != r.SolutionID) {
		return nil, mapRoleDomainError(domainerr.New(constant.CodeRoleNotFound))
	}
	return uc.replacePermissions(ctx, r, req.PermissionCodes)
}

// ── Helper ───────────────────────────────────────────────────────────────────

func (uc *RoleUseCase) create(ctx context.Context, p entity.NewRoleParams) (*dto.RoleResponse, error) {
	r, err := entity.NewRole(p, uc.now())
	if err != nil {
		return nil, mapRoleDomainError(err)
	}
	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Roles.Save(ctx, r); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx, roleEvent(event.TypeRoleCreated, r, nil))
	})
	if err != nil {
		return nil, mapRoleDomainError(err)
	}
	resp := toRoleResponse(r)
	return &resp, nil
}

func (uc *RoleUseCase) replacePermissions(ctx context.Context, r *entity.Role, codes []string) ([]dto.RolePermissionResponse, error) {
	if err := r.EnsureEditable(); err != nil {
		return nil, mapRoleDomainError(err)
	}
	err := uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		catalog, err := uc.deps.Permissions.ListBySolution(ctx, r.SolutionID)
		if err != nil {
			return err
		}
		current, err := uc.deps.RolePermissions.PermissionIDs(ctx, r.ID)
		if err != nil {
			return err
		}
		resolved, err := roleservice.ResolvePermissions(codes, catalog, current)
		if err != nil {
			return err
		}
		ids := make([]string, 0, len(resolved))
		resolvedCodes := make([]string, 0, len(resolved))
		for _, p := range resolved {
			ids = append(ids, p.ID)
			resolvedCodes = append(resolvedCodes, p.Code.Value())
		}
		if err := uc.deps.RolePermissions.ReplaceAll(ctx, r.ID, r.SolutionID, ids); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx, roleEvent(event.TypeRolePermissionsChanged, r, resolvedCodes))
	})
	if err != nil {
		return nil, mapRoleDomainError(err)
	}
	return uc.permissions(ctx, r.ID)
}

func (uc *RoleUseCase) list(ctx context.Context, q port.RoleListReadQuery) ([]dto.RoleResponse, dto.Meta, error) {
	items, meta, err := uc.deps.Query.ListRoles(ctx, q)
	if err != nil {
		return nil, dto.Meta{}, mapRoleDomainError(err)
	}
	out := make([]dto.RoleResponse, 0, len(items))
	for _, it := range items {
		out = append(out, dto.RoleResponse{
			ID: it.ID, SolutionID: it.SolutionID, TenantID: it.TenantID, Code: it.Code, Name: it.Name,
			Description: it.Description, IsSystem: it.IsSystem, Scope: roleScope(it.TenantID, it.IsSystem),
			CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt,
		})
	}
	return out, meta, nil
}

func (uc *RoleUseCase) permissions(ctx context.Context, roleID string) ([]dto.RolePermissionResponse, error) {
	items, err := uc.deps.Query.ListRolePermissions(ctx, roleID)
	if err != nil {
		return nil, mapRoleDomainError(err)
	}
	out := make([]dto.RolePermissionResponse, 0, len(items))
	for _, it := range items {
		out = append(out, dto.RolePermissionResponse{ID: it.ID, Code: it.Code, Name: it.Name, Status: it.Status})
	}
	return out, nil
}

// ensureSolutionVisible — solution harus ada; solution inactive disembunyikan dari non-admin.
func (uc *RoleUseCase) ensureSolutionVisible(ctx context.Context, actor *principal.Principal, solutionID string) error {
	sol, err := uc.deps.Solutions.FindByID(ctx, solutionID)
	if err != nil {
		return mapRoleDomainError(err)
	}
	if !sol.IsActive() && !isPlatformAdmin(actor) {
		return mapRoleDomainError(domainerr.New(solutionconstant.CodeSolutionNotFound))
	}
	return nil
}

// ensureTenantSolutionAccess — token service hanya untuk solution miliknya; akses tenant
// via AccessPolicy; tenant & solution harus ada.
func (uc *RoleUseCase) ensureTenantSolutionAccess(ctx context.Context, actor *principal.Principal, tenantID, solutionID string) error {
	if actor.IsService() && actor.SolutionID != solutionID {
		return mapRoleDomainError(domainerr.New(tenantconstant.CodeTenantAccessDenied))
	}
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), tenantID); err != nil {
		return mapRoleDomainError(err)
	}
	if _, err := uc.deps.Tenants.FindByID(ctx, tenantID); err != nil {
		return mapRoleDomainError(err)
	}
	return uc.ensureSolutionVisible(ctx, actor, solutionID)
}

// ensureTenantRoleAccess — akses custom role lewat ID: ditolak → 404.
func (uc *RoleUseCase) ensureTenantRoleAccess(ctx context.Context, actor *principal.Principal, tenantID string, r *entity.Role) error {
	notFound := mapRoleDomainError(domainerr.New(constant.CodeRoleNotFound))
	if actor.IsService() && actor.SolutionID != r.SolutionID {
		return notFound
	}
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), tenantID); err != nil {
		if isDomainCode(err, tenantconstant.CodeTenantAccessDenied) {
			return notFound
		}
		return mapRoleDomainError(err)
	}
	return nil
}
