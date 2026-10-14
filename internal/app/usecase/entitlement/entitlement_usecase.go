package entitlement

import (
	"context"
	"math/big"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/principal"
	benefitentity "platform-service/internal/domain/benefit/entity"
	benefitrepo "platform-service/internal/domain/benefit/repository"
	entservice "platform-service/internal/domain/entitlement/service"
	domainerr "platform-service/internal/domain/errors"
	solutionconstant "platform-service/internal/domain/solution/constant"
	solutionrepo "platform-service/internal/domain/solution/repository"
	subrepo "platform-service/internal/domain/subscription/repository"
	tenantconstant "platform-service/internal/domain/tenant/constant"
	tenantrepo "platform-service/internal/domain/tenant/repository"
	tenantservice "platform-service/internal/domain/tenant/service"
	usagerepo "platform-service/internal/domain/usage/repository"
	usageservice "platform-service/internal/domain/usage/service"
)

type EntitlementDeps struct {
	Benefits      benefitrepo.BenefitRepository
	Solutions     solutionrepo.SolutionRepository
	Tenants       tenantrepo.TenantRepository
	Members       tenantrepo.TenantMemberRepository
	Subscriptions subrepo.SubscriptionRepository
	Resolver      *entservice.Resolver
	TenantUsage   usagerepo.TenantUsageRepository
}

type EntitlementUseCase struct {
	deps   EntitlementDeps
	access *tenantservice.AccessPolicy
	now    func() time.Time
}

func NewEntitlementUseCase(deps EntitlementDeps) *EntitlementUseCase {
	return &EntitlementUseCase{deps: deps, access: tenantservice.NewAccessPolicy(deps.Members, deps.Subscriptions), now: time.Now}
}

// List menghitung entitlement tenant atas seluruh benefit solution dari snapshot
// subscription (spec §3.10), lengkap dengan limit / used / remaining untuk benefit di-meter.
//
// Required — auth: user (platform admin / member dengan tid) atau service | scope: entitlements:read
func (uc *EntitlementUseCase) List(ctx context.Context, actor *principal.Principal, tenantID, solutionID string) (*dto.EntitlementsResponse, error) {
	if err := uc.authorize(ctx, actor, tenantID, solutionID); err != nil {
		return nil, err
	}
	benefits, err := uc.deps.Benefits.ListBySolution(ctx, solutionID)
	if err != nil {
		return nil, mapEntitlementDomainError(err)
	}
	return uc.evaluate(ctx, tenantID, solutionID, benefits)
}

// Get menghitung entitlement satu benefit.
//
// Required — auth: user (platform admin / member dengan tid) atau service | scope: entitlements:read
func (uc *EntitlementUseCase) Get(ctx context.Context, actor *principal.Principal, tenantID, solutionID, benefitCode string) (*dto.EntitlementResponse, error) {
	if err := uc.authorize(ctx, actor, tenantID, solutionID); err != nil {
		return nil, err
	}
	b, err := uc.deps.Benefits.FindByCode(ctx, solutionID, benefitCode)
	if err != nil {
		return nil, mapEntitlementDomainError(err)
	}
	resp, err := uc.evaluate(ctx, tenantID, solutionID, []*benefitentity.Benefit{b})
	if err != nil {
		return nil, err
	}
	return &resp.Entitlements[0], nil
}

// authorize: token service hanya untuk solution miliknya; tenant & solution harus ada;
// akses tenant mengikuti AccessPolicy.
func (uc *EntitlementUseCase) authorize(ctx context.Context, actor *principal.Principal, tenantID, solutionID string) error {
	if actor.IsService() && actor.SolutionID != solutionID {
		return mapEntitlementDomainError(domainerr.New(tenantconstant.CodeTenantAccessDenied))
	}
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), tenantID); err != nil {
		return mapEntitlementDomainError(err)
	}
	if _, err := uc.deps.Tenants.FindByID(ctx, tenantID); err != nil {
		return mapEntitlementDomainError(err)
	}
	sol, err := uc.deps.Solutions.FindByID(ctx, solutionID)
	if err != nil {
		return mapEntitlementDomainError(err)
	}
	if !sol.IsActive() && !isPlatformAdmin(actor) {
		return mapEntitlementDomainError(domainerr.New(solutionconstant.CodeSolutionNotFound))
	}
	return nil
}

func (uc *EntitlementUseCase) evaluate(ctx context.Context, tenantID, solutionID string, benefits []*benefitentity.Benefit) (*dto.EntitlementsResponse, error) {
	src, err := uc.deps.Resolver.Resolve(ctx, tenantID, solutionID)
	if err != nil {
		return nil, mapEntitlementDomainError(err)
	}
	resp := &dto.EntitlementsResponse{TenantID: tenantID, SolutionID: solutionID, Entitlements: make([]dto.EntitlementResponse, 0, len(benefits))}
	var billingStart, billingEnd *time.Time
	if s := src.Subscription; s != nil {
		resp.Subscription = &dto.EntitlementSubscription{ID: s.ID, Status: s.Status.String(), PlanID: s.PlanID, CurrentPeriodEnd: s.CurrentPeriodEnd}
		billingStart, billingEnd = s.CurrentPeriodStart, s.CurrentPeriodEnd
	}

	now := uc.now()
	for _, b := range benefits {
		ent := entservice.Calculate(b, src.ByBenefit[b.ID])
		item := toEntitlementResponse(ent)

		if b.IsMetered() && ent.Granted {
			period := usageservice.ResolvePeriod(b.UsageReset, now, billingStart, billingEnd)
			tu, err := uc.deps.TenantUsage.Get(ctx, tenantID, b.ID, period.Start)
			if err != nil {
				return nil, mapEntitlementDomainError(err)
			}
			used := new(big.Rat)
			if tu != nil {
				used.SetString(tu.Used)
			}
			usedStr := entservice.FormatRat(used)
			item.Used = &usedStr
			start := period.Start
			item.PeriodStart, item.PeriodEnd = &start, period.End
			if ent.Limit != nil {
				rem, reached := entservice.Remaining(ent.Limit, used)
				remStr := entservice.FormatRat(rem)
				item.Remaining = &remStr
				item.LimitReached = reached
			}
		}
		resp.Entitlements = append(resp.Entitlements, item)
	}
	return resp, nil
}
