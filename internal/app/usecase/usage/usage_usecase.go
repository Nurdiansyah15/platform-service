package usage

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	benefitentity "platform-service/internal/domain/benefit/entity"
	benefitrepo "platform-service/internal/domain/benefit/repository"
	benefitvo "platform-service/internal/domain/benefit/valueobject"
	entevent "platform-service/internal/domain/entitlement/event"
	entservice "platform-service/internal/domain/entitlement/service"
	domainerr "platform-service/internal/domain/errors"
	clientrepo "platform-service/internal/domain/solutionclient/repository"
	subrepo "platform-service/internal/domain/subscription/repository"
	tenantrepo "platform-service/internal/domain/tenant/repository"
	tenantservice "platform-service/internal/domain/tenant/service"
	"platform-service/internal/domain/usage/constant"
	"platform-service/internal/domain/usage/entity"
	usagerepo "platform-service/internal/domain/usage/repository"
	usageservice "platform-service/internal/domain/usage/service"
	"platform-service/internal/domain/usage/valueobject"
)

type UsageDeps struct {
	Tx            port.Transactor
	Outbox        port.EventOutbox
	Resolver      *entservice.Resolver
	Records       usagerepo.UsageRecordRepository
	TenantUsage   usagerepo.TenantUsageRepository
	Benefits      benefitrepo.BenefitRepository
	Clients       clientrepo.SolutionClientRepository
	Subscriptions subrepo.SubscriptionRepository
	Members       tenantrepo.TenantMemberRepository
}

type UsageUseCase struct {
	deps   UsageDeps
	access *tenantservice.AccessPolicy
	now    func() time.Time
}

func NewUsageUseCase(deps UsageDeps) *UsageUseCase {
	return &UsageUseCase{deps: deps, access: tenantservice.NewAccessPolicy(deps.Members, deps.Subscriptions), now: time.Now}
}

// ReportDelta mencatat pemakaian benefit counter (used = used + quantity).
// Idempotent per (solution, idempotency_key).
//
// Required — auth: service | scope: usage:write
func (uc *UsageUseCase) ReportDelta(ctx context.Context, actor *principal.Principal, tenantID, solutionID string, req dto.ReportUsageDeltaRequest) (*dto.UsageReportResponse, error) {
	return uc.report(ctx, actor, tenantID, solutionID, req.BenefitCode, valueobject.ModeDelta, req.Quantity, req.IdempotencyKey, req.OccurredAt)
}

// ReportAbsolute mencatat nilai pemakaian terkini benefit gauge (used = quantity).
// Laporan dengan occurred_at lebih lama dari yang sudah diterapkan diabaikan.
//
// Required — auth: service | scope: usage:write
func (uc *UsageUseCase) ReportAbsolute(ctx context.Context, actor *principal.Principal, tenantID, solutionID, benefitCode string, req dto.ReportUsageAbsoluteRequest) (*dto.UsageReportResponse, error) {
	return uc.report(ctx, actor, tenantID, solutionID, benefitCode, valueobject.ModeAbsolute, req.Quantity, req.IdempotencyKey, req.OccurredAt)
}

func (uc *UsageUseCase) report(ctx context.Context, actor *principal.Principal, tenantID, solutionID, benefitCode string,
	mode valueobject.Mode, rawQty json.RawMessage, rawKey string, occurredAt *time.Time) (*dto.UsageReportResponse, error) {

	// Client hanya boleh melaporkan usage solution miliknya, untuk tenant yang subscribe.
	if actor.SolutionID != solutionID {
		return nil, mapUsageDomainError(domainerr.New(constant.CodeUsageSolutionMismatch))
	}
	if err := uc.access.EnsureCanAccess(ctx, tenantservice.Accessor{ServiceSolutionID: actor.SolutionID}, tenantID); err != nil {
		return nil, mapUsageDomainError(domainerr.New(constant.CodeUsageNoSubscription))
	}

	b, err := uc.deps.Benefits.FindByCode(ctx, solutionID, benefitCode)
	if err != nil {
		return nil, mapUsageDomainError(err)
	}
	qty, err := valueobject.ParseQuantityJSON(rawQty)
	if err != nil {
		return nil, mapUsageDomainError(err)
	}
	key, err := valueobject.NewIdempotencyKey(rawKey)
	if err != nil {
		return nil, mapUsageDomainError(err)
	}
	client, err := uc.deps.Clients.FindByClientID(ctx, actor.ClientID)
	if err != nil {
		return nil, mapUsageDomainError(err)
	}

	now := uc.now()
	params := entity.NewUsageRecordParams{
		TenantID: tenantID, Benefit: b, Mode: mode, Quantity: qty, IdempotencyKey: key, ReportedByClientID: client.ID,
	}
	if occurredAt != nil {
		params.OccurredAt = *occurredAt
	}
	record, err := entity.NewUsageRecord(params, now)
	if err != nil {
		return nil, mapUsageDomainError(err)
	}
	period, err := uc.period(ctx, b, tenantID, record.OccurredAt)
	if err != nil {
		return nil, mapUsageDomainError(err)
	}
	// Limit efektif benefit ini (untuk event usage.limit_reached).
	src, err := uc.deps.Resolver.Resolve(ctx, tenantID, solutionID)
	if err != nil {
		return nil, mapUsageDomainError(err)
	}
	limit := entservice.Calculate(b, src.ByBenefit[b.ID]).Limit

	var current *entity.TenantUsage
	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Records.Insert(ctx, record); err != nil {
			return err
		}
		previous, err := uc.previousUsed(ctx, record, period)
		if err != nil {
			return err
		}
		if mode == valueobject.ModeDelta {
			current, err = uc.deps.TenantUsage.AddDelta(ctx, record, period)
		} else {
			current, err = uc.deps.TenantUsage.SetAbsolute(ctx, record, period)
		}
		if err != nil {
			return err
		}
		return uc.emitLimitReached(ctx, b, record, current, previous, limit)
	})
	if err != nil {
		if isDomainCode(err, constant.CodeUsageDuplicateKey) {
			return uc.replay(ctx, record, b, limit)
		}
		return nil, mapUsageDomainError(err)
	}
	return toResponse(b, record, current, limit, false), nil
}

// replay menangani idempotency_key yang sudah dipakai: payload identik → kembalikan
// pemakaian terkini tanpa menerapkan ulang; payload berbeda → konflik.
func (uc *UsageUseCase) replay(ctx context.Context, attempted *entity.UsageRecord, b *benefitentity.Benefit, limit *big.Rat) (*dto.UsageReportResponse, error) {
	original, err := uc.deps.Records.FindByIdempotencyKey(ctx, attempted.SolutionID, attempted.IdempotencyKey.Value())
	if err != nil {
		return nil, mapUsageDomainError(err)
	}
	if !original.SamePayload(attempted) {
		return nil, mapUsageDomainError(domainerr.New(constant.CodeUsageIdempotencyConflict))
	}
	period, err := uc.period(ctx, b, original.TenantID, original.OccurredAt)
	if err != nil {
		return nil, mapUsageDomainError(err)
	}
	current, err := uc.deps.TenantUsage.Get(ctx, original.TenantID, original.BenefitID, period.Start)
	if err != nil {
		return nil, mapUsageDomainError(err)
	}
	if current == nil {
		current = &entity.TenantUsage{Period: period, Used: "0.0000"}
	}
	return toResponse(b, original, current, limit, true), nil
}

// previousUsed membaca pemakaian sebelum laporan diterapkan (di dalam transaksi).
func (uc *UsageUseCase) previousUsed(ctx context.Context, r *entity.UsageRecord, period entity.Period) (*big.Rat, error) {
	prev := new(big.Rat)
	tu, err := uc.deps.TenantUsage.Get(ctx, r.TenantID, r.BenefitID, period.Start)
	if err != nil {
		return nil, err
	}
	if tu != nil {
		prev.SetString(tu.Used)
	}
	return prev, nil
}

// emitLimitReached menulis usage.limit_reached hanya saat pemakaian melewati limit
// (sebelumnya < limit, sekarang >= limit), agar tidak dikirim berulang setiap laporan.
func (uc *UsageUseCase) emitLimitReached(ctx context.Context, b *benefitentity.Benefit, r *entity.UsageRecord, current *entity.TenantUsage, previous, limit *big.Rat) error {
	if limit == nil {
		return nil
	}
	used, ok := new(big.Rat).SetString(current.Used)
	if !ok || previous.Cmp(limit) >= 0 || used.Cmp(limit) < 0 {
		return nil
	}
	return uc.deps.Outbox.Append(ctx, port.OutboxEvent{
		EventType:  entevent.TypeUsageLimitReached,
		TenantID:   &r.TenantID,
		SolutionID: &r.SolutionID,
		Payload: entevent.UsageLimitReachedPayload{
			TenantID: r.TenantID, SolutionID: r.SolutionID, BenefitCode: b.Code.Value(),
			Limit: entservice.FormatRat(limit), Used: entservice.FormatRat(used),
			PeriodStart: current.Period.Start, PeriodEnd: current.Period.End, OccurredAt: r.OccurredAt,
		},
	})
}

func (uc *UsageUseCase) period(ctx context.Context, b *benefitentity.Benefit, tenantID string, occurredAt time.Time) (entity.Period, error) {
	var start, end *time.Time
	if b.UsageReset == benefitvo.UsageResetBillingPeriod {
		s, err := uc.deps.Subscriptions.FindUsable(ctx, tenantID, b.SolutionID)
		if err != nil {
			return entity.Period{}, err
		}
		start, end = s.CurrentPeriodStart, s.CurrentPeriodEnd
	}
	return usageservice.ResolvePeriod(b.UsageReset, occurredAt, start, end), nil
}
