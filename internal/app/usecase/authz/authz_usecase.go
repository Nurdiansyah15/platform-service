package authz

import (
	"context"
	"sort"
	"strings"
	"time"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	"platform-service/internal/domain/authz/constant"
	"platform-service/internal/domain/authz/entity"
	"platform-service/internal/domain/authz/event"
	authzrepo "platform-service/internal/domain/authz/repository"
	authzservice "platform-service/internal/domain/authz/service"
	"platform-service/internal/domain/authz/valueobject"
	domainerr "platform-service/internal/domain/errors"
	orgconstant "platform-service/internal/domain/organization/constant"
	orgrepo "platform-service/internal/domain/organization/repository"
	orgservice "platform-service/internal/domain/organization/service"
	permrepo "platform-service/internal/domain/permission/repository"
	roleconstant "platform-service/internal/domain/role/constant"
	rolerepo "platform-service/internal/domain/role/repository"
	solutionrepo "platform-service/internal/domain/solution/repository"
	subrepo "platform-service/internal/domain/subscription/repository"
)

type AuthzDeps struct {
	Tx            port.Transactor
	Outbox        port.EventOutbox
	Assignments   authzrepo.RoleAssignmentRepository
	Grants        authzrepo.GrantReader
	Scopes        *authzservice.ScopeValidator
	Roles         rolerepo.RoleRepository
	Members       orgrepo.CompanyMemberRepository
	Subscriptions subrepo.SubscriptionRepository
	Solutions     solutionrepo.SolutionRepository
	Query         port.RoleAssignmentQueryReadModel
	// RolePermissions — scope_type yang dipakai role (nilai scope wajib saat assign).
	RolePermissions rolerepo.RolePermissionRepository
	// Permissions & PermissionGroups — info tampilan (nama, grup) untuk effective permissions.
	Permissions      permrepo.PermissionRepository
	PermissionGroups permrepo.PermissionGroupRepository
	// Admin menegakkan permission platform role.assign untuk token user.
	Admin *authzservice.CompanyAdminPolicy
}

type AuthzUseCase struct {
	deps   AuthzDeps
	access *orgservice.AccessPolicy
	now    func() time.Time
}

func NewAuthzUseCase(deps AuthzDeps) *AuthzUseCase {
	return &AuthzUseCase{deps: deps, access: orgservice.NewAccessPolicy(deps.Members, deps.Subscriptions), now: time.Now}
}

// AssignRole memberi role ke user dalam company (spec §2.6): user member active, custom role
// milik company, dan scope_values berisi tepat satu nilai valid untuk setiap scope_type
// non-company yang dipakai permission role (branch milik company; scope custom terdaftar &
// aktif). Role yang sama dengan nilai berbeda = assignment lain.
//
// Required — auth: user (platform admin / member dengan permission role.assign di scope tsb) atau service milik solution role | scope: roles:write
func (uc *AuthzUseCase) AssignRole(ctx context.Context, actor *principal.Principal, companyID, userID string, req dto.CreateRoleAssignmentRequest) (*dto.RoleAssignmentResponse, error) {
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), companyID); err != nil {
		return nil, mapAuthzDomainError(err)
	}
	if active, err := uc.isActiveMember(ctx, companyID, userID); err != nil {
		return nil, mapAuthzDomainError(err)
	} else if !active {
		return nil, mapAuthzDomainError(domainerr.New(constant.CodeAssignmentNotMember))
	}

	r, err := uc.deps.Roles.FindByID(ctx, req.RoleID)
	if err != nil {
		if isDomainCode(err, roleconstant.CodeRoleNotFound) {
			return nil, mapAuthzDomainError(domainerr.New(constant.CodeAssignmentRoleUnavailable))
		}
		return nil, mapAuthzDomainError(err)
	}
	// Custom role hanya bisa di-assign di company pemilik; token service hanya role solution-nya.
	if (!r.IsGlobal() && !r.IsOwnedBy(companyID)) || (actor.IsService() && actor.SolutionID != r.SolutionID) {
		return nil, mapAuthzDomainError(domainerr.New(constant.CodeAssignmentRoleUnavailable))
	}
	// GATE-SUBSCRIPTION (nonaktif sementara): company tidak wajib subscribe solution role.
	// usable, err := uc.deps.Subscriptions.HasUsable(ctx, companyID, r.SolutionID)
	// if err != nil {
	// 	return nil, mapAuthzDomainError(err)
	// }
	// if !usable {
	// 	return nil, mapAuthzDomainError(domainerr.New(constant.CodeAssignmentNotSubscribed))
	// }

	roleScopes, err := uc.deps.RolePermissions.ScopeTypes(ctx, r.ID)
	if err != nil {
		return nil, mapAuthzDomainError(err)
	}
	values, labels, err := uc.resolveScopeValues(ctx, r.SolutionID, companyID, roleScopes, req.ScopeValues, req.ScopeLabels)
	if err != nil {
		return nil, mapAuthzDomainError(err)
	}
	if err := uc.ensureCanAssign(ctx, actor, companyID, r.SolutionID, assignScope(companyID, roleScopes, values)); err != nil {
		return nil, err
	}

	a := entity.NewRoleAssignment(companyID, userID, r.ID, values, labels, uc.now())
	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Assignments.Create(ctx, a); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx, port.OutboxEvent{
			EventType:  event.TypeRoleAssignmentCreated,
			CompanyID:  &companyID,
			SolutionID: &r.SolutionID,
			Payload: event.RoleAssignmentPayload{
				ID: a.ID, CompanyID: companyID, UserID: userID, RoleID: r.ID, RoleCode: r.Code.Value(),
				SolutionID: r.SolutionID, ScopeValues: a.Scopes, ScopeLabels: a.Labels, CreatedAt: a.CreatedAt,
			},
		})
	})
	if err != nil {
		return nil, mapAuthzDomainError(err)
	}
	return &dto.RoleAssignmentResponse{
		ID: a.ID, CompanyID: companyID, UserID: userID, RoleID: r.ID, RoleCode: r.Code.Value(), SolutionID: r.SolutionID,
		ScopeValues: a.Scopes, ScopeLabels: a.Labels, CreatedAt: a.CreatedAt,
	}, nil
}

// ListAssignments menampilkan role assignment user di company. Token service hanya melihat
// role milik solution-nya; token user boleh memfilter dengan solution_id.
//
// Required — auth: user (platform admin / member dengan cid) atau service | scope: roles:write
func (uc *AuthzUseCase) ListAssignments(ctx context.Context, actor *principal.Principal, companyID, userID string, req dto.SolutionFilterQuery) ([]dto.RoleAssignmentDetailResponse, error) {
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), companyID); err != nil {
		return nil, mapAuthzDomainError(err)
	}
	solutionID := req.SolutionID
	if actor.IsService() {
		solutionID = &actor.SolutionID
	}
	items, err := uc.deps.Query.ListForMember(ctx, port.RoleAssignmentListReadQuery{CompanyID: companyID, UserID: userID, SolutionID: solutionID})
	if err != nil {
		return nil, mapAuthzDomainError(err)
	}
	return items, nil
}

// RevokeAssignment mencabut role assignment. Assignment milik company yang tidak bisa diakses
// (atau role solution lain untuk token service) dilaporkan 404.
//
// Required — auth: user (platform admin / member dengan permission role.assign di scope tsb) atau service | scope: roles:write
func (uc *AuthzUseCase) RevokeAssignment(ctx context.Context, actor *principal.Principal, assignmentID string) error {
	notFound := mapAuthzDomainError(domainerr.New(constant.CodeAssignmentNotFound))
	a, err := uc.deps.Assignments.FindByID(ctx, assignmentID)
	if err != nil {
		return mapAuthzDomainError(err)
	}
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), a.CompanyID); err != nil {
		if isDomainCode(err, orgconstant.CodeCompanyAccessDenied) {
			return notFound
		}
		return mapAuthzDomainError(err)
	}
	if actor.IsService() && actor.SolutionID != a.Role.SolutionID {
		return notFound
	}
	roleScopes, err := uc.deps.RolePermissions.ScopeTypes(ctx, a.RoleID)
	if err != nil {
		return mapAuthzDomainError(err)
	}
	if err := uc.ensureCanAssign(ctx, actor, a.CompanyID, a.Role.SolutionID, assignScope(a.CompanyID, roleScopes, a.Scopes)); err != nil {
		return err
	}

	now := uc.now()
	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Assignments.Delete(ctx, a.ID); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx, port.OutboxEvent{
			EventType:  event.TypeRoleAssignmentDeleted,
			CompanyID:  &a.CompanyID,
			SolutionID: &a.Role.SolutionID,
			Payload: event.RoleAssignmentPayload{
				ID: a.ID, CompanyID: a.CompanyID, UserID: a.UserID, RoleID: a.RoleID, RoleCode: a.Role.Code,
				SolutionID: a.Role.SolutionID, ScopeValues: a.Scopes, ScopeLabels: a.Labels, CreatedAt: a.CreatedAt,
			},
			OccurredAt: now,
		})
	})
	if err != nil {
		return mapAuthzDomainError(err)
	}
	return nil
}

// EffectivePermissions menampilkan seluruh permission user di company untuk satu solution,
// beserta role & scope asalnya. Token user wajib mengisi solution_id; token service memakai
// solution client. User bukan member active → daftar kosong.
//
// Required — auth: user (platform admin / member dengan cid) atau service | scope: authz:check
func (uc *AuthzUseCase) EffectivePermissions(ctx context.Context, actor *principal.Principal, companyID, userID string, req dto.SolutionFilterQuery) ([]dto.EffectivePermissionResponse, error) {
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), companyID); err != nil {
		return nil, mapAuthzDomainError(err)
	}
	solutionID := actor.SolutionID
	if !actor.IsService() {
		if req.SolutionID == nil {
			return nil, mapAuthzDomainError(domainerr.New(constant.CodeSolutionRequired))
		}
		solutionID = *req.SolutionID
	}
	if _, err := uc.deps.Solutions.FindByID(ctx, solutionID); err != nil {
		return nil, mapAuthzDomainError(err)
	}

	grants, err := uc.effectiveGrants(ctx, companyID, userID, solutionID)
	if err != nil {
		return nil, mapAuthzDomainError(err)
	}
	catalog, err := uc.permissionCatalog(ctx, solutionID)
	if err != nil {
		return nil, mapAuthzDomainError(err)
	}
	byPermission := map[string]*dto.EffectivePermissionResponse{}
	for _, g := range grants {
		p, ok := byPermission[g.PermissionCode]
		if !ok {
			p = &dto.EffectivePermissionResponse{Permission: g.PermissionCode, Name: g.PermissionCode}
			if info, found := catalog[g.PermissionCode]; found {
				p.Name, p.Group = info.Name, info.Group
			}
			byPermission[g.PermissionCode] = p
		}
		p.Grants = append(p.Grants, dto.EffectiveGrantResponse{RoleCode: g.RoleCode, ScopeType: g.Scope.Type.String(), ScopeID: g.Scope.ID})
	}
	out := make([]dto.EffectivePermissionResponse, 0, len(byPermission))
	for _, p := range byPermission {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Permission < out[j].Permission })
	return out, nil
}

// Check memeriksa satu permission.
//
// Required — auth: service (authz:check, solution miliknya) atau user (diri sendiri, tid wajib) | scope: authz:check
func (uc *AuthzUseCase) Check(ctx context.Context, actor *principal.Principal, req dto.AuthzCheckRequest) (*dto.AuthzCheckResult, error) {
	resp, err := uc.CheckBatch(ctx, actor, dto.AuthzCheckBatchRequest{
		CompanyID: req.CompanyID, UserID: req.UserID,
		Checks: []dto.AuthzCheckItem{{Permission: req.Permission, ScopeType: req.ScopeType, ScopeID: req.ScopeID, ParentID: req.ParentID}},
	})
	if err != nil {
		return nil, err
	}
	return &resp.Results[0], nil
}

// CheckBatch memeriksa beberapa permission sekaligus untuk satu user & company.
// Tidak diizinkan (allowed=false) bila member tidak active atau subscription tidak usable.
//
// Required — auth: service (authz:check, solution miliknya) atau user (diri sendiri, tid wajib) | scope: authz:check
func (uc *AuthzUseCase) CheckBatch(ctx context.Context, actor *principal.Principal, req dto.AuthzCheckBatchRequest) (*dto.AuthzCheckBatchResponse, error) {
	companyID, userID, solutionID, err := uc.resolveTarget(ctx, actor, req.CompanyID, req.UserID)
	if err != nil {
		return nil, err
	}

	// Validasi semua scope lebih dulu agar request yang salah ditolak utuh (422).
	scopes := make([]entity.CheckScope, len(req.Checks))
	for i, c := range req.Checks {
		st, err := valueobject.NewScopeType(c.ScopeType)
		if err != nil {
			return nil, mapAuthzDomainError(err)
		}
		if scopes[i], err = uc.deps.Scopes.Resolve(ctx, authzservice.ScopeInput{
			SolutionID: solutionID, CompanyID: companyID, Type: st, ID: c.ScopeID, ParentID: c.ParentID,
		}); err != nil {
			return nil, mapAuthzDomainError(err)
		}
	}

	grants, err := uc.effectiveGrants(ctx, companyID, userID, solutionID)
	if err != nil {
		return nil, mapAuthzDomainError(err)
	}

	resp := &dto.AuthzCheckBatchResponse{CompanyID: companyID, UserID: userID, Results: make([]dto.AuthzCheckResult, 0, len(req.Checks))}
	for i, c := range req.Checks {
		allowed, roles := authzservice.Evaluate(grants, c.Permission, scopes[i])
		if roles == nil {
			roles = []string{}
		}
		resp.Results = append(resp.Results, dto.AuthzCheckResult{
			Permission: c.Permission, ScopeType: scopes[i].Scope.Type.String(), ScopeID: scopes[i].Scope.ID,
			Allowed: allowed, GrantedBy: roles,
		})
	}
	return resp, nil
}

// resolveTarget menentukan company, user, dan solution yang dicek:
//   - service: company_id & user_id wajib, solution = milik client
//   - user: diri sendiri pada company aktif token, solution = aud token
func (uc *AuthzUseCase) resolveTarget(ctx context.Context, actor *principal.Principal, companyID, userID string) (string, string, string, error) {
	if actor.IsService() {
		if companyID == "" || userID == "" {
			return "", "", "", mapAuthzDomainError(domainerr.New(constant.CodeCheckTargetRequired))
		}
		return companyID, userID, actor.SolutionID, nil
	}

	if !actor.HasActiveCompany() {
		return "", "", "", apperror.Forbidden(apperror.MsgCompanyNotSelected)
	}
	if (companyID != "" && companyID != actor.CompanyID) || (userID != "" && userID != actor.UserID) {
		return "", "", "", mapAuthzDomainError(domainerr.New(orgconstant.CodeCompanyAccessDenied))
	}
	sol, err := uc.deps.Solutions.FindByCode(ctx, actor.SolutionCode)
	if err != nil {
		return "", "", "", mapAuthzDomainError(domainerr.New(constant.CodeCheckSolutionUnavailable))
	}
	return actor.CompanyID, actor.UserID, sol.ID, nil
}

// effectiveGrants — kosong bila user bukan member active atau company tidak subscribe (usable).
func (uc *AuthzUseCase) effectiveGrants(ctx context.Context, companyID, userID, solutionID string) ([]entity.Grant, error) {
	active, err := uc.isActiveMember(ctx, companyID, userID)
	if err != nil || !active {
		return nil, err
	}
	// GATE-SUBSCRIPTION (nonaktif sementara): grant tidak dikosongkan bila company belum subscribe.
	// usable, err := uc.deps.Subscriptions.HasUsable(ctx, companyID, solutionID)
	// if err != nil || !usable {
	// 	return nil, err
	// }
	return uc.deps.Grants.Grants(ctx, companyID, userID, solutionID)
}

// ensureCanAssign — token user wajib punya role.assign pada scope assignment; role solution
// console (permission platform) hanya untuk pemegang seluruh permission platform di scope itu.
func (uc *AuthzUseCase) ensureCanAssign(ctx context.Context, actor *principal.Principal, companyID, roleSolutionID string, scope entity.CheckScope) error {
	a := accessorOf(actor)
	if err := uc.deps.Admin.EnsurePermission(ctx, a, companyID, constant.PermRoleAssign, scope); err != nil {
		return mapAuthzDomainError(err)
	}
	if err := uc.deps.Admin.EnsurePlatformRoleControl(ctx, a, companyID, roleSolutionID, scope); err != nil {
		return mapAuthzDomainError(err)
	}
	return nil
}

func (uc *AuthzUseCase) isActiveMember(ctx context.Context, companyID, userID string) (bool, error) {
	m, err := uc.deps.Members.Find(ctx, companyID, userID)
	if err != nil {
		if isDomainCode(err, orgconstant.CodeMemberNotFound) {
			return false, nil
		}
		return false, err
	}
	return m.IsActive(), nil
}

// permissionInfo — info tampilan permission dari katalog solution.
type permissionInfo struct {
	Name  string
	Group *dto.PermissionGroupRef
}

func (uc *AuthzUseCase) permissionCatalog(ctx context.Context, solutionID string) (map[string]permissionInfo, error) {
	perms, err := uc.deps.Permissions.ListBySolution(ctx, solutionID)
	if err != nil {
		return nil, err
	}
	groups, err := uc.deps.PermissionGroups.ListBySolution(ctx, solutionID)
	if err != nil {
		return nil, err
	}
	groupByID := make(map[string]*dto.PermissionGroupRef, len(groups))
	for _, g := range groups {
		groupByID[g.ID] = &dto.PermissionGroupRef{Code: g.Code.Value(), Name: g.Name, SortOrder: g.SortOrder}
	}
	out := make(map[string]permissionInfo, len(perms))
	for _, p := range perms {
		info := permissionInfo{Name: p.Name}
		if p.GroupID != nil {
			info.Group = groupByID[*p.GroupID]
		}
		out[p.Code.Value()] = info
	}
	return out, nil
}

// resolveScopeValues memvalidasi nilai scope assignment terhadap scope_type yang dipakai role:
// setiap scope_type non-company wajib punya nilai valid, kunci lain ditolak. Label hanya
// disimpan untuk scope custom.
func (uc *AuthzUseCase) resolveScopeValues(ctx context.Context, solutionID, companyID string, roleScopes []string,
	rawValues, rawLabels map[string]string) (map[string]string, map[string]string, error) {
	required := map[string]bool{}
	for _, t := range roleScopes {
		if t != valueobject.ScopeCompany.String() {
			required[t] = true
		}
	}
	for k := range rawValues {
		if !required[k] {
			return nil, nil, domainerr.New(constant.CodeAssignmentScopeUnexpected)
		}
	}
	for k := range rawLabels {
		if !required[k] {
			return nil, nil, domainerr.New(constant.CodeAssignmentScopeUnexpected)
		}
	}

	values := make(map[string]string, len(required))
	labels := map[string]string{}
	for t := range required {
		v := strings.TrimSpace(rawValues[t])
		if v == "" {
			return nil, nil, domainerr.New(constant.CodeAssignmentScopeMissing)
		}
		st, err := valueobject.NewScopeType(t)
		if err != nil {
			return nil, nil, err
		}
		resolved, err := uc.deps.Scopes.Resolve(ctx, authzservice.ScopeInput{
			SolutionID: solutionID, CompanyID: companyID, Type: st, ID: v, ForAssignment: true,
		})
		if err != nil {
			return nil, nil, err
		}
		values[t] = resolved.Scope.ID
		if l := strings.TrimSpace(rawLabels[t]); l != "" && !st.IsBuiltin() {
			if len([]rune(l)) > constant.ScopeLabelMaxLength {
				return nil, nil, domainerr.New(constant.CodeScopeInvalid)
			}
			labels[t] = l
		}
	}
	return values, labels, nil
}

// assignScope — scope tempat pemberi role wajib memegang role.assign: level company bila role
// memberi permission ber-scope company atau scope custom; level branch bila seluruh
// permission role ber-scope branch (cabang dari nilai assignment).
func assignScope(companyID string, roleScopes []string, values map[string]string) entity.CheckScope {
	for _, t := range roleScopes {
		if t != valueobject.ScopeBranch.String() {
			return authzservice.CompanyScope(companyID)
		}
	}
	for k := range values {
		if k != valueobject.ScopeBranch.String() {
			return authzservice.CompanyScope(companyID)
		}
	}
	if b := values[valueobject.ScopeBranch.String()]; b != "" {
		return authzservice.BranchScope(companyID, b)
	}
	return authzservice.CompanyScope(companyID)
}
