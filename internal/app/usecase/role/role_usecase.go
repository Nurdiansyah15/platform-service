package role

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	authzconstant "platform-service/internal/domain/authz/constant"
	authzrepo "platform-service/internal/domain/authz/repository"
	authzservice "platform-service/internal/domain/authz/service"
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
	// Assignments — assignment role ikut dihapus saat role dihapus.
	Assignments authzrepo.RoleAssignmentRepository
	// Admin menegakkan permission platform role.manage untuk token user.
	Admin *authzservice.CompanyAdminPolicy
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

// GetGlobal mengambil detail role lewat ID. Custom role hanya untuk company pemilik (selain itu 404).
//
// Required — auth: user / service | scope: -
func (uc *RoleUseCase) GetGlobal(ctx context.Context, actor *principal.Principal, roleID string) (*dto.RoleResponse, error) {
	r, err := uc.visibleRole(ctx, actor, roleID)
	if err != nil {
		return nil, err
	}
	return uc.detail(ctx, r, r.CompanyID)
}

// UpdateGlobal mengubah nama / deskripsi role global. Role sistem tidak bisa diubah.
//
// Required — auth: user (platform admin) | scope: -
func (uc *RoleUseCase) UpdateGlobal(ctx context.Context, roleID string, req dto.UpdateRoleRequest) (*dto.RoleResponse, error) {
	r, err := uc.globalRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	return uc.update(ctx, r, req, nil)
}

// DeleteGlobal menghapus role global (soft delete) beserta seluruh assignment-nya.
//
// Required — auth: user (platform admin) | scope: -
func (uc *RoleUseCase) DeleteGlobal(ctx context.Context, roleID string) error {
	r, err := uc.globalRole(ctx, roleID)
	if err != nil {
		return err
	}
	return uc.delete(ctx, r)
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
	return uc.list(ctx, port.RoleListReadQuery{SolutionID: solutionID, CompanyID: &companyID, Search: req.Search, Kind: req.Kind, PaginationParams: req.PaginationParams})
}

// CreateForCompany membuat custom role milik company. Company wajib subscribe solution
// (trialing / active) — GATE-SUBSCRIPTION (nonaktif sementara).
//
// Required — auth: user (platform admin / member dengan permission role.manage) atau service milik solution | scope: roles:write
func (uc *RoleUseCase) CreateForCompany(ctx context.Context, actor *principal.Principal, companyID, solutionID string, req dto.CreateRoleRequest) (*dto.RoleResponse, error) {
	if err := uc.ensureCompanySolutionAccess(ctx, actor, companyID, solutionID); err != nil {
		return nil, err
	}
	if err := uc.ensureRoleManage(ctx, actor, companyID, solutionID); err != nil {
		return nil, err
	}
	// GATE-SUBSCRIPTION (nonaktif sementara): company tidak wajib subscribe solution.
	// ok, err := uc.deps.Subscriptions.HasUsable(ctx, companyID, solutionID)
	// if err != nil {
	// 	return nil, mapRoleDomainError(err)
	// }
	// if !ok {
	// 	return nil, mapRoleDomainError(domainerr.New(constant.CodeRoleCompanyNotSubscribed))
	// }
	return uc.create(ctx, entity.NewRoleParams{SolutionID: solutionID, CompanyID: &companyID, Code: req.Code, Name: req.Name, Description: req.Description})
}

// GetForCompany mengambil role (global atau custom milik company) dalam konteks company;
// jumlah pemegang hanya dihitung di company tersebut.
//
// Required — auth: user (platform admin / member dengan cid) atau service milik solution | scope: roles:write
func (uc *RoleUseCase) GetForCompany(ctx context.Context, actor *principal.Principal, companyID, roleID string) (*dto.RoleResponse, error) {
	r, err := uc.companyVisibleRole(ctx, actor, companyID, roleID)
	if err != nil {
		return nil, err
	}
	return uc.detail(ctx, r, &companyID)
}

// ListHolders menampilkan pemegang role di company (user + scope assignment).
//
// Required — auth: user (platform admin / member dengan cid) atau service milik solution | scope: roles:write
func (uc *RoleUseCase) ListHolders(ctx context.Context, actor *principal.Principal, companyID, roleID string) ([]dto.RoleHolderResponse, error) {
	r, err := uc.companyVisibleRole(ctx, actor, companyID, roleID)
	if err != nil {
		return nil, err
	}
	items, err := uc.deps.Query.ListHolders(ctx, r.ID, companyID)
	if err != nil {
		return nil, mapRoleDomainError(err)
	}
	return items, nil
}

// UpdateForCompany mengubah nama / deskripsi custom role milik company.
//
// Required — auth: user (platform admin / member dengan permission role.manage) atau service milik solution | scope: roles:write
func (uc *RoleUseCase) UpdateForCompany(ctx context.Context, actor *principal.Principal, companyID, roleID string, req dto.UpdateRoleRequest) (*dto.RoleResponse, error) {
	r, err := uc.manageableRole(ctx, actor, companyID, roleID)
	if err != nil {
		return nil, err
	}
	return uc.update(ctx, r, req, &companyID)
}

// DeleteForCompany menghapus custom role milik company (soft delete) beserta assignment-nya.
//
// Required — auth: user (platform admin / member dengan permission role.manage) atau service milik solution | scope: roles:write
func (uc *RoleUseCase) DeleteForCompany(ctx context.Context, actor *principal.Principal, companyID, roleID string) error {
	r, err := uc.manageableRole(ctx, actor, companyID, roleID)
	if err != nil {
		return err
	}
	return uc.delete(ctx, r)
}

// ReplaceCompanyRolePermissions mengganti permission custom role milik company. Role global
// atau milik company lain → 404.
//
// Required — auth: user (platform admin / member dengan permission role.manage) atau service milik solution | scope: roles:write
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
	if err := uc.ensureRoleManage(ctx, actor, companyID, r.SolutionID); err != nil {
		return nil, err
	}
	return uc.replacePermissions(ctx, r, req.PermissionCodes)
}

// ── Helper ───────────────────────────────────────────────────────────────────

// visibleRole — role lewat ID: global harus solution-nya terlihat; custom hanya company pemilik.
func (uc *RoleUseCase) visibleRole(ctx context.Context, actor *principal.Principal, roleID string) (*entity.Role, error) {
	r, err := uc.deps.Roles.FindByID(ctx, roleID)
	if err != nil {
		return nil, mapRoleDomainError(err)
	}
	if r.IsGlobal() {
		if err := uc.ensureSolutionVisible(ctx, actor, r.SolutionID); err != nil {
			return nil, mapRoleDomainError(domainerr.New(constant.CodeRoleNotFound))
		}
		return r, nil
	}
	if err := uc.ensureCompanyRoleAccess(ctx, actor, *r.CompanyID, r); err != nil {
		return nil, err
	}
	return r, nil
}

// companyVisibleRole — role global atau custom milik company, dilihat dari company tersebut.
func (uc *RoleUseCase) companyVisibleRole(ctx context.Context, actor *principal.Principal, companyID, roleID string) (*entity.Role, error) {
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), companyID); err != nil {
		return nil, mapRoleDomainError(err)
	}
	r, err := uc.deps.Roles.FindByID(ctx, roleID)
	if err != nil {
		return nil, mapRoleDomainError(err)
	}
	if (!r.IsGlobal() && !r.IsOwnedBy(companyID)) || (actor.IsService() && actor.SolutionID != r.SolutionID) {
		return nil, mapRoleDomainError(domainerr.New(constant.CodeRoleNotFound))
	}
	return r, nil
}

// manageableRole — ownedRole + permission platform role.manage.
func (uc *RoleUseCase) manageableRole(ctx context.Context, actor *principal.Principal, companyID, roleID string) (*entity.Role, error) {
	r, err := uc.ownedRole(ctx, actor, companyID, roleID)
	if err != nil {
		return nil, err
	}
	if err := uc.ensureRoleManage(ctx, actor, companyID, r.SolutionID); err != nil {
		return nil, err
	}
	return r, nil
}

// ensureRoleManage — token user wajib punya role.manage; custom role solution console
// (berisi permission platform) hanya untuk pemegang seluruh permission platform.
func (uc *RoleUseCase) ensureRoleManage(ctx context.Context, actor *principal.Principal, companyID, solutionID string) error {
	a, scope := accessorOf(actor), authzservice.CompanyScope(companyID)
	if err := uc.deps.Admin.EnsurePermission(ctx, a, companyID, authzconstant.PermRoleManage, scope); err != nil {
		return mapRoleDomainError(err)
	}
	if err := uc.deps.Admin.EnsurePlatformRoleControl(ctx, a, companyID, solutionID, scope); err != nil {
		return mapRoleDomainError(err)
	}
	return nil
}

// ownedRole — custom role milik company (role global / company lain → 404).
func (uc *RoleUseCase) ownedRole(ctx context.Context, actor *principal.Principal, companyID, roleID string) (*entity.Role, error) {
	r, err := uc.companyVisibleRole(ctx, actor, companyID, roleID)
	if err != nil {
		return nil, err
	}
	if !r.IsOwnedBy(companyID) {
		return nil, mapRoleDomainError(domainerr.New(constant.CodeRoleNotFound))
	}
	return r, nil
}

// globalRole — role global lewat ID (custom role → 404).
func (uc *RoleUseCase) globalRole(ctx context.Context, roleID string) (*entity.Role, error) {
	r, err := uc.deps.Roles.FindByID(ctx, roleID)
	if err != nil {
		return nil, mapRoleDomainError(err)
	}
	if !r.IsGlobal() {
		return nil, mapRoleDomainError(domainerr.New(constant.CodeRoleNotFound))
	}
	return r, nil
}

func (uc *RoleUseCase) detail(ctx context.Context, r *entity.Role, companyID *string) (*dto.RoleResponse, error) {
	st, err := uc.deps.Query.Stats(ctx, r.ID, companyID)
	if err != nil {
		return nil, mapRoleDomainError(err)
	}
	resp := toRoleResponse(r)
	resp.PermissionCount, resp.MemberCount = st.PermissionCount, st.MemberCount
	return &resp, nil
}

func (uc *RoleUseCase) update(ctx context.Context, r *entity.Role, req dto.UpdateRoleRequest, companyID *string) (*dto.RoleResponse, error) {
	if err := r.Update(req.Name, req.Description, uc.now()); err != nil {
		return nil, mapRoleDomainError(err)
	}
	err := uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Roles.Save(ctx, r); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx, roleEvent(event.TypeRoleUpdated, r, nil))
	})
	if err != nil {
		return nil, mapRoleDomainError(err)
	}
	return uc.detail(ctx, r, companyID)
}

func (uc *RoleUseCase) delete(ctx context.Context, r *entity.Role) error {
	if err := r.Delete(uc.now()); err != nil {
		return mapRoleDomainError(err)
	}
	err := uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Roles.Save(ctx, r); err != nil {
			return err
		}
		if err := uc.deps.Assignments.DeleteByRole(ctx, r.ID); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx, roleEvent(event.TypeRoleDeleted, r, nil))
	})
	if err != nil {
		return mapRoleDomainError(err)
	}
	return nil
}

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
			CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt, PermissionCount: it.PermissionCount, MemberCount: it.MemberCount,
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
