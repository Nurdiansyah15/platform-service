package authz

import (
	"context"
	"sort"
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

// AssignRole memberi role ke user dalam company pada scope tertentu (spec §2.6):
// user member active, company subscribe solution role, custom role milik company,
// scope_id valid untuk company.
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

	scopeType, err := valueobject.NewScopeType(req.ScopeType)
	if err != nil {
		return nil, mapAuthzDomainError(err)
	}
	scope, err := uc.deps.Scopes.Resolve(ctx, companyID, scopeType, req.ScopeID)
	if err != nil {
		return nil, mapAuthzDomainError(err)
	}
	if err := uc.ensureCanAssign(ctx, actor, companyID, r.SolutionID, scope); err != nil {
		return nil, err
	}

	a := entity.NewRoleAssignment(companyID, userID, r.ID, scope.Scope, uc.now())
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
				SolutionID: r.SolutionID, ScopeType: a.Scope.Type.String(), ScopeID: a.Scope.ID, CreatedAt: a.CreatedAt,
			},
		})
	})
	if err != nil {
		return nil, mapAuthzDomainError(err)
	}
	return &dto.RoleAssignmentResponse{
		ID: a.ID, CompanyID: companyID, UserID: userID, RoleID: r.ID, RoleCode: r.Code.Value(), SolutionID: r.SolutionID,
		ScopeType: a.Scope.Type.String(), ScopeID: a.Scope.ID, CreatedAt: a.CreatedAt,
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
	check := entity.CheckScope{Scope: a.Scope}
	if a.Scope.Type == valueobject.ScopeBranch {
		check.BranchCompanyID = a.CompanyID
	}
	if err := uc.ensureCanAssign(ctx, actor, a.CompanyID, a.Role.SolutionID, check); err != nil {
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
				SolutionID: a.Role.SolutionID, ScopeType: a.Scope.Type.String(), ScopeID: a.Scope.ID, CreatedAt: a.CreatedAt,
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
	byPermission := map[string]*dto.EffectivePermissionResponse{}
	for _, g := range grants {
		p, ok := byPermission[g.PermissionCode]
		if !ok {
			p = &dto.EffectivePermissionResponse{Permission: g.PermissionCode}
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
		Checks: []dto.AuthzCheckItem{{Permission: req.Permission, ScopeType: req.ScopeType, ScopeID: req.ScopeID}},
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
		if scopes[i], err = uc.deps.Scopes.Resolve(ctx, companyID, st, c.ScopeID); err != nil {
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
