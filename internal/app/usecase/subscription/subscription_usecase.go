package subscription

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	addonrepo "platform-service/internal/domain/addon/repository"
	domainerr "platform-service/internal/domain/errors"
	planrepo "platform-service/internal/domain/plan/repository"
	solutionrepo "platform-service/internal/domain/solution/repository"
	"platform-service/internal/domain/subscription/constant"
	"platform-service/internal/domain/subscription/entity"
	"platform-service/internal/domain/subscription/event"
	subrepo "platform-service/internal/domain/subscription/repository"
	subservice "platform-service/internal/domain/subscription/service"
	tenantconstant "platform-service/internal/domain/tenant/constant"
	tenantrepo "platform-service/internal/domain/tenant/repository"
	tenantservice "platform-service/internal/domain/tenant/service"
)

type SubscriptionDeps struct {
	Tx                 port.Transactor
	Outbox             port.EventOutbox
	Subscriptions      subrepo.SubscriptionRepository
	SubscriptionAddons subrepo.SubscriptionAddonRepository
	Snapshots          subrepo.SnapshotRepository
	BenefitValues      subservice.BenefitValueSource
	Tenants            tenantrepo.TenantRepository
	Members            tenantrepo.TenantMemberRepository
	Plans              planrepo.PlanRepository
	Addons             addonrepo.AddonRepository
	Solutions          solutionrepo.SolutionRepository
	Query              port.SubscriptionQueryReadModel
}

type SubscriptionUseCase struct {
	deps      SubscriptionDeps
	access    *tenantservice.AccessPolicy
	snapshots *subservice.SnapshotBuilder
	now       func() time.Time
}

func NewSubscriptionUseCase(deps SubscriptionDeps) *SubscriptionUseCase {
	return &SubscriptionUseCase{
		deps:      deps,
		access:    tenantservice.NewAccessPolicy(deps.Members, deps.Subscriptions),
		snapshots: subservice.NewSnapshotBuilder(deps.BenefitValues),
		now:       time.Now,
	}
}

// Create membuat subscription tenant ke plan dan membekukan benefit plan ke snapshot.
//
// Required — auth: user (platform admin) | scope: - | deployment: cloud (LICENSE_MANAGED di on_prem)
func (uc *SubscriptionUseCase) Create(ctx context.Context, tenantID string, req dto.CreateSubscriptionRequest) (*dto.SubscriptionResponse, error) {
	tenant, err := uc.deps.Tenants.FindByID(ctx, tenantID)
	if err != nil {
		return nil, mapSubscriptionDomainError(err)
	}
	if !tenant.IsActive() {
		return nil, mapSubscriptionDomainError(domainerr.New(constant.CodeSubscriptionTenantInactive))
	}
	terms, err := uc.planTerms(ctx, req.PlanID)
	if err != nil {
		return nil, mapSubscriptionDomainError(err)
	}

	now := uc.now()
	periodStart := now
	if req.CurrentPeriodStart != nil {
		periodStart = *req.CurrentPeriodStart
	}
	s, err := entity.NewSubscription(tenantID, terms, req.Status, periodStart, now)
	if err != nil {
		return nil, mapSubscriptionDomainError(err)
	}

	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Subscriptions.Save(ctx, s); err != nil {
			return err
		}
		if err := uc.replacePlanSnapshots(ctx, s, now); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx,
			subscriptionEvent(event.TypeSubscriptionCreated, s, "", ""),
			entitlementChanged(s, "subscription_created", now))
	})
	if err != nil {
		return nil, mapSubscriptionDomainError(err)
	}
	return uc.get(ctx, s.ID)
}

// List menampilkan subscription tenant. Token service hanya melihat subscription solution miliknya.
//
// Required — auth: user (platform admin / member dengan tid) atau service | scope: entitlements:read
func (uc *SubscriptionUseCase) List(ctx context.Context, actor *principal.Principal, tenantID string, req dto.ListSubscriptionQuery) ([]dto.SubscriptionResponse, dto.Meta, error) {
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), tenantID); err != nil {
		return nil, dto.Meta{}, mapSubscriptionDomainError(err)
	}
	if _, err := uc.deps.Tenants.FindByID(ctx, tenantID); err != nil {
		return nil, dto.Meta{}, mapSubscriptionDomainError(err)
	}
	q := port.SubscriptionListReadQuery{TenantID: tenantID, Status: req.Status, PaginationParams: req.PaginationParams}
	if actor.IsService() {
		q.SolutionID = &actor.SolutionID
	}
	items, meta, err := uc.deps.Query.ListSubscriptions(ctx, q)
	if err != nil {
		return nil, dto.Meta{}, mapSubscriptionDomainError(err)
	}
	out := make([]dto.SubscriptionResponse, 0, len(items))
	for _, it := range items {
		out = append(out, toSubscriptionResponse(it))
	}
	return out, meta, nil
}

// ChangePlan mengganti plan subscription live dan mengganti snapshot benefit plan.
//
// Required — auth: user (platform admin) | scope: - | deployment: cloud (LICENSE_MANAGED di on_prem)
func (uc *SubscriptionUseCase) ChangePlan(ctx context.Context, subscriptionID string, req dto.ChangePlanRequest) (*dto.SubscriptionResponse, error) {
	s, err := uc.deps.Subscriptions.FindByID(ctx, subscriptionID)
	if err != nil {
		return nil, mapSubscriptionDomainError(err)
	}
	terms, err := uc.planTerms(ctx, req.PlanID)
	if err != nil {
		return nil, mapSubscriptionDomainError(err)
	}
	previousPlan := s.PlanID
	now := uc.now()
	if err := s.ChangePlan(terms, now); err != nil {
		return nil, mapSubscriptionDomainError(err)
	}

	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Subscriptions.Save(ctx, s); err != nil {
			return err
		}
		if err := uc.replacePlanSnapshots(ctx, s, now); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx,
			subscriptionEvent(event.TypeSubscriptionPlanChanged, s, previousPlan, ""),
			entitlementChanged(s, "plan_changed", now))
	})
	if err != nil {
		return nil, mapSubscriptionDomainError(err)
	}
	return uc.get(ctx, s.ID)
}

// Cancel membatalkan subscription live. Snapshot dipertahankan sebagai histori.
//
// Required — auth: user (platform admin) | scope: - | deployment: cloud (LICENSE_MANAGED di on_prem)
func (uc *SubscriptionUseCase) Cancel(ctx context.Context, subscriptionID string) (*dto.SubscriptionResponse, error) {
	s, err := uc.deps.Subscriptions.FindByID(ctx, subscriptionID)
	if err != nil {
		return nil, mapSubscriptionDomainError(err)
	}
	previousStatus := s.Status.String()
	now := uc.now()
	if err := s.Cancel(now); err != nil {
		return nil, mapSubscriptionDomainError(err)
	}

	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Subscriptions.Save(ctx, s); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx,
			subscriptionEvent(event.TypeSubscriptionStatusChanged, s, "", previousStatus),
			entitlementChanged(s, "subscription_canceled", now))
	})
	if err != nil {
		return nil, mapSubscriptionDomainError(err)
	}
	return uc.get(ctx, s.ID)
}

// ListAddons menampilkan add-on subscription. Subscription yang tidak boleh diakses dilaporkan 404.
//
// Required — auth: user (platform admin / member dengan tid) atau service | scope: entitlements:read
func (uc *SubscriptionUseCase) ListAddons(ctx context.Context, actor *principal.Principal, subscriptionID string) ([]dto.SubscriptionAddonResponse, error) {
	s, err := uc.accessibleSubscription(ctx, actor, subscriptionID)
	if err != nil {
		return nil, err
	}
	items, err := uc.deps.Query.ListSubscriptionAddons(ctx, s.ID)
	if err != nil {
		return nil, mapSubscriptionDomainError(err)
	}
	out := make([]dto.SubscriptionAddonResponse, 0, len(items))
	for _, it := range items {
		out = append(out, toSubscriptionAddonResponse(it))
	}
	return out, nil
}

// AddAddon memasang add-on ke subscription live dan membekukan benefit add-on ke snapshot.
// Add-on yang pernah dibatalkan diaktifkan ulang.
//
// Required — auth: user (platform admin) | scope: - | deployment: cloud (LICENSE_MANAGED di on_prem)
func (uc *SubscriptionUseCase) AddAddon(ctx context.Context, subscriptionID string, req dto.AddSubscriptionAddonRequest) (*dto.SubscriptionAddonResponse, error) {
	s, err := uc.deps.Subscriptions.FindByID(ctx, subscriptionID)
	if err != nil {
		return nil, mapSubscriptionDomainError(err)
	}
	terms, err := uc.addonTerms(ctx, req.AddonID)
	if err != nil {
		return nil, mapSubscriptionDomainError(err)
	}
	quantity := 1
	if req.Quantity != nil {
		quantity = *req.Quantity
	}
	now := uc.now()

	var sa *entity.SubscriptionAddon
	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		existing, err := uc.deps.SubscriptionAddons.Find(ctx, s.ID, terms.AddonID)
		switch {
		case err == nil:
			if !s.Status.IsLive() {
				return domainerr.New(constant.CodeSubscriptionNotLive)
			}
			if err := existing.Reactivate(terms, quantity, now); err != nil {
				return err
			}
			sa = existing
		case isDomainCode(err, constant.CodeSubscriptionAddonNotFound):
			if sa, err = s.AttachAddon(terms, quantity, now); err != nil {
				return err
			}
		default:
			return err
		}
		if err := uc.deps.SubscriptionAddons.Save(ctx, sa); err != nil {
			return err
		}
		rows, err := uc.snapshots.ForAddon(ctx, sa, now)
		if err != nil {
			return err
		}
		if err := uc.deps.Snapshots.ReplaceAddonSnapshots(ctx, sa.ID, rows); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx,
			subscriptionAddonEvent(s, sa),
			entitlementChanged(s, "addon_added", now))
	})
	if err != nil {
		return nil, mapSubscriptionDomainError(err)
	}

	item, err := uc.deps.Query.GetSubscriptionAddon(ctx, sa.ID)
	if err != nil {
		return nil, mapSubscriptionDomainError(err)
	}
	resp := toSubscriptionAddonResponse(*item)
	return &resp, nil
}

// accessibleSubscription memuat subscription lalu memeriksa akses ke tenant pemiliknya.
// Akses ditolak / solution lain (token service) dilaporkan 404.
func (uc *SubscriptionUseCase) accessibleSubscription(ctx context.Context, actor *principal.Principal, id string) (*entity.Subscription, error) {
	s, err := uc.deps.Subscriptions.FindByID(ctx, id)
	if err != nil {
		return nil, mapSubscriptionDomainError(err)
	}
	notFound := mapSubscriptionDomainError(domainerr.New(constant.CodeSubscriptionNotFound))
	if actor.IsService() && actor.SolutionID != s.SolutionID {
		return nil, notFound
	}
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), s.TenantID); err != nil {
		if isDomainCode(err, tenantconstant.CodeTenantAccessDenied) {
			return nil, notFound
		}
		return nil, mapSubscriptionDomainError(err)
	}
	return s, nil
}

func (uc *SubscriptionUseCase) replacePlanSnapshots(ctx context.Context, s *entity.Subscription, now time.Time) error {
	rows, err := uc.snapshots.ForPlan(ctx, s, now)
	if err != nil {
		return err
	}
	return uc.deps.Snapshots.ReplacePlanSnapshots(ctx, s.ID, rows)
}

// planTerms — plan harus ada, aktif, dan solution-nya aktif.
func (uc *SubscriptionUseCase) planTerms(ctx context.Context, planID string) (entity.PlanTerms, error) {
	unavailable := domainerr.New(constant.CodeSubscriptionPlanUnavailable)
	p, err := uc.deps.Plans.FindByID(ctx, planID)
	if err != nil || !p.IsActive {
		return entity.PlanTerms{}, unavailable
	}
	sol, err := uc.deps.Solutions.FindByID(ctx, p.SolutionID)
	if err != nil || !sol.IsActive() {
		return entity.PlanTerms{}, unavailable
	}
	return entity.PlanTerms{PlanID: p.ID, SolutionID: p.SolutionID, BillingCycle: p.BillingCycle, Price: p.Price, Currency: p.Currency}, nil
}

// addonTerms — add-on harus ada dan aktif (kecocokan solution dicek entity).
func (uc *SubscriptionUseCase) addonTerms(ctx context.Context, addonID string) (entity.AddonTerms, error) {
	a, err := uc.deps.Addons.FindByID(ctx, addonID)
	if err != nil || !a.IsActive {
		return entity.AddonTerms{}, domainerr.New(constant.CodeSubscriptionAddonUnavailable)
	}
	return entity.AddonTerms{AddonID: a.ID, SolutionID: a.SolutionID, Price: a.Price, Currency: a.Currency}, nil
}

func (uc *SubscriptionUseCase) get(ctx context.Context, id string) (*dto.SubscriptionResponse, error) {
	item, err := uc.deps.Query.GetSubscription(ctx, id)
	if err != nil {
		return nil, mapSubscriptionDomainError(err)
	}
	resp := toSubscriptionResponse(*item)
	return &resp, nil
}
