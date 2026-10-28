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
	orgconstant "platform-service/internal/domain/organization/constant"
	orgrepo "platform-service/internal/domain/organization/repository"
	orgservice "platform-service/internal/domain/organization/service"
	solutionconstant "platform-service/internal/domain/solution/constant"
	solutionrepo "platform-service/internal/domain/solution/repository"
	subrepo "platform-service/internal/domain/subscription/repository"
	usagerepo "platform-service/internal/domain/usage/repository"
	usageservice "platform-service/internal/domain/usage/service"
)

type EntitlementDeps struct {
	Benefits      benefitrepo.BenefitRepository
	Solutions     solutionrepo.SolutionRepository
	Companies     orgrepo.CompanyRepository
	Members       orgrepo.CompanyMemberRepository
	Subscriptions subrepo.SubscriptionRepository
	Resolver      *entservice.Resolver
	CompanyUsage  usagerepo.CompanyUsageRepository
}

type EntitlementUseCase struct {
	deps   EntitlementDeps
	access *orgservice.AccessPolicy
	now    func() time.Time
}

func NewEntitlementUseCase(deps EntitlementDeps) *EntitlementUseCase {
	return &EntitlementUseCase{deps: deps, access: orgservice.NewAccessPolicy(deps.Members, deps.Subscriptions), now: time.Now}
}

// List menghitung entitlement company atas seluruh benefit solution dari snapshot
// subscription (spec §3.10), lengkap dengan limit / used / remaining untuk benefit di-meter.
//
// Required — auth: user (platform admin / member dengan tid) atau service | scope: entitlements:read
func (uc *EntitlementUseCase) List(ctx context.Context, actor *principal.Principal, companyID, solutionID string) (*dto.EntitlementsResponse, error) {
	if err := uc.authorize(ctx, actor, companyID, solutionID); err != nil {
		return nil, err
	}
	benefits, err := uc.deps.Benefits.ListBySolution(ctx, solutionID)
	if err != nil {
		return nil, mapEntitlementDomainError(err)
	}
	return uc.evaluate(ctx, companyID, solutionID, benefits)
}

// Get menghitung entitlement satu benefit.
//
// Required — auth: user (platform admin / member dengan tid) atau service | scope: entitlements:read
func (uc *EntitlementUseCase) Get(ctx context.Context, actor *principal.Principal, companyID, solutionID, benefitCode string) (*dto.EntitlementResponse, error) {
	if err := uc.authorize(ctx, actor, companyID, solutionID); err != nil {
		return nil, err
	}
	b, err := uc.deps.Benefits.FindByCode(ctx, solutionID, benefitCode)
	if err != nil {
		return nil, mapEntitlementDomainError(err)
	}
	resp, err := uc.evaluate(ctx, companyID, solutionID, []*benefitentity.Benefit{b})
	if err != nil {
		return nil, err
	}
	return &resp.Entitlements[0], nil
}

// authorize: token service hanya untuk solution miliknya; company & solution harus ada;
// akses company mengikuti AccessPolicy.
func (uc *EntitlementUseCase) authorize(ctx context.Context, actor *principal.Principal, companyID, solutionID string) error {
	if actor.IsService() && actor.SolutionID != solutionID {
		return mapEntitlementDomainError(domainerr.New(orgconstant.CodeCompanyAccessDenied))
	}
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), companyID); err != nil {
		return mapEntitlementDomainError(err)
	}
	if _, err := uc.deps.Companies.FindByID(ctx, companyID); err != nil {
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

func (uc *EntitlementUseCase) evaluate(ctx context.Context, companyID, solutionID string, benefits []*benefitentity.Benefit) (*dto.EntitlementsResponse, error) {
	src, err := uc.deps.Resolver.Resolve(ctx, companyID, solutionID)
	if err != nil {
		return nil, mapEntitlementDomainError(err)
	}
	resp := &dto.EntitlementsResponse{CompanyID: companyID, SolutionID: solutionID, Entitlements: make([]dto.EntitlementResponse, 0, len(benefits))}
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
			tu, err := uc.deps.CompanyUsage.Get(ctx, companyID, b.ID, period.Start)
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
