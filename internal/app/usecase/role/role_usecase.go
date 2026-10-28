package role

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	domainerr "platform-service/internal/domain/errors"
	orgconstant "platform-service/internal/domain/organization/constant"
	orgrepo "platform-service/internal/domain/organization/repository"
	orgservice "platform-service/internal/domain/organization/service"
	permrepo "platform-service/internal/domain/permission/repository"
	"platform-service/internal/domain/role/constant"
	"platform-service/internal/domain/role/entity"
	"platform-service/internal/domain/role/event"
	rolerepo "platform-service/internal/domain/role/repository"
	roleservice "platform-service/internal/domain/role/service"
	solutionconstant "platform-service/internal/domain/solution/constant"
	solutionrepo "platform-service/internal/domain/solution/repository"
	subrepo "platform-service/internal/domain/subscription/repository"
)

type RoleDeps struct {
	Tx              port.Transactor
	Outbox          port.EventOutbox
	Roles           rolerepo.RoleRepository
	RolePermissions rolerepo.RolePermissionRepository
	Permissions     permrepo.PermissionRepository
	Solutions       solutionrepo.SolutionRepository
	Companies       orgrepo.CompanyRepository
	Members         orgrepo.CompanyMemberRepository
	Subscriptions   subrepo.SubscriptionRepository
	Query           port.RoleQueryReadModel
}

type RoleUseCase struct {
	deps   RoleDeps
	access *orgservice.AccessPolicy
	now    func() time.Time
}

func NewRoleUseCase(deps RoleDeps) *RoleUseCase {
	return &RoleUseCase{deps: deps, access: orgservice.NewAccessPolicy(deps.Members, deps.Subscriptions), now: time.Now}
}

// ── Role global (platform) ───────────────────────────────────────────────────

// ListGlobal menampilkan role global solution (company_id NULL).
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

// ListPermissions menampilkan permission role. Custom role hanya untuk company pemilik
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
	} else if err := uc.ensureCompanyRoleAccess(ctx, actor, *r.CompanyID, r); err != nil {
		return nil, err
	}
	return uc.permissions(ctx, r.ID)
}

// ReplaceGlobalPermissions mengganti seluruh permission role global. Role sistem tidak
// bisa diubah; custom role company harus lewat endpoint company (404 di sini).
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

// ── Role custom company ───────────────────────────────────────────────────────

// ListForCompany menampilkan role global + custom role milik company untuk solution.
//
// Required — auth: user (platform admin / member dengan tid) atau service milik solution | scope: roles:write
func (uc *RoleUseCase) ListForCompany(ctx context.Context, actor *principal.Principal, companyID, solutionID string, req dto.ListRoleQuery) ([]dto.RoleResponse, dto.Meta, error) {
	if err := uc.ensureCompanySolutionAccess(ctx, actor, companyID, solutionID); err != nil {
		return nil, dto.Meta{}, err
	}
	return uc.list(ctx, port.RoleListReadQuery{SolutionID: solutionID, CompanyID: &companyID, Search: req.Search, PaginationParams: req.PaginationParams})
}

// CreateForCompany membuat custom role milik company. Company wajib subscribe solution
// (trialing / active).
//
// Required — auth: user (platform admin / member dengan tid) atau service milik solution | scope: roles:write
func (uc *RoleUseCase) CreateForCompany(ctx context.Context, actor *principal.Principal, companyID, solutionID string, req dto.CreateRoleRequest) (*dto.RoleResponse, error) {
	if err := uc.ensureCompanySolutionAccess(ctx, actor, companyID, solutionID); err != nil {
		return nil, err
	}
	ok, err := uc.deps.Subscriptions.HasUsable(ctx, companyID, solutionID)
	if err != nil {
		return nil, mapRoleDomainError(err)
	}
	if !ok {
		return nil, mapRoleDomainError(domainerr.New(constant.CodeRoleCompanyNotSubscribed))
	}
	return uc.create(ctx, entity.NewRoleParams{SolutionID: solutionID, CompanyID: &companyID, Code: req.Code, Name: req.Name, Description: req.Description})
}

// ReplaceCompanyRolePermissions mengganti permission custom role milik company. Role global
// atau milik company lain → 404.
//
// Required — auth: user (platform admin / member dengan tid) atau service milik solution | scope: roles:write
func (uc *RoleUseCase) ReplaceCompanyRolePermissions(ctx context.Context, actor *principal.Principal, companyID, roleID string, req dto.ReplaceRolePermissionsRequest) ([]dto.RolePermissionResponse, error) {
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), companyID); err != nil {
		return nil, mapRoleDomainError(err)
	}
	r, err := uc.deps.Roles.FindByID(ctx, roleID)
	if err != nil {
		return nil, mapRoleDomainError(err)
	}
	if !r.IsOwnedBy(companyID) || (actor.IsService() && actor.SolutionID != r.SolutionID) {
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
			ID: it.ID, SolutionID: it.SolutionID, CompanyID: it.CompanyID, Code: it.Code, Name: it.Name,
			Description: it.Description, IsSystem: it.IsSystem, Scope: roleScope(it.CompanyID, it.IsSystem),
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

// ensureCompanySolutionAccess — token service hanya untuk solution miliknya; akses company
// via AccessPolicy; company & solution harus ada.
func (uc *RoleUseCase) ensureCompanySolutionAccess(ctx context.Context, actor *principal.Principal, companyID, solutionID string) error {
	if actor.IsService() && actor.SolutionID != solutionID {
		return mapRoleDomainError(domainerr.New(orgconstant.CodeCompanyAccessDenied))
	}
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), companyID); err != nil {
		return mapRoleDomainError(err)
	}
	if _, err := uc.deps.Companies.FindByID(ctx, companyID); err != nil {
		return mapRoleDomainError(err)
	}
	return uc.ensureSolutionVisible(ctx, actor, solutionID)
}

// ensureCompanyRoleAccess — akses custom role lewat ID: ditolak → 404.
func (uc *RoleUseCase) ensureCompanyRoleAccess(ctx context.Context, actor *principal.Principal, companyID string, r *entity.Role) error {
	notFound := mapRoleDomainError(domainerr.New(constant.CodeRoleNotFound))
	if actor.IsService() && actor.SolutionID != r.SolutionID {
		return notFound
	}
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), companyID); err != nil {
		if isDomainCode(err, orgconstant.CodeCompanyAccessDenied) {
			return notFound
		}
		return mapRoleDomainError(err)
	}
	return nil
}
