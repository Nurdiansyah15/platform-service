package authz

import (
	"context"
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
// Required — auth: user (platform admin / member dengan tid) atau service milik solution role | scope: roles:write
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
	usable, err := uc.deps.Subscriptions.HasUsable(ctx, companyID, r.SolutionID)
	if err != nil {
		return nil, mapAuthzDomainError(err)
	}
	if !usable {
		return nil, mapAuthzDomainError(domainerr.New(constant.CodeAssignmentNotSubscribed))
	}

	scopeType, err := valueobject.NewScopeType(req.ScopeType)
	if err != nil {
		return nil, mapAuthzDomainError(err)
	}
	scope, err := uc.deps.Scopes.Resolve(ctx, companyID, scopeType, req.ScopeID)
	if err != nil {
		return nil, mapAuthzDomainError(err)
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
	usable, err := uc.deps.Subscriptions.HasUsable(ctx, companyID, solutionID)
	if err != nil || !usable {
		return nil, err
	}
	return uc.deps.Grants.Grants(ctx, companyID, userID, solutionID)
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
