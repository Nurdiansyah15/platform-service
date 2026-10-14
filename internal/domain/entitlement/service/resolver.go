package service

import (
	"context"
	"errors"

	"platform-service/internal/domain/entitlement/entity"
	"platform-service/internal/domain/entitlement/repository"
	domainerr "platform-service/internal/domain/errors"
	subconstant "platform-service/internal/domain/subscription/constant"
	subentity "platform-service/internal/domain/subscription/entity"
	subrepo "platform-service/internal/domain/subscription/repository"
)

// Resolver mengumpulkan bahan perhitungan entitlement tenant untuk satu solution:
// subscription usable (trialing / active) dan snapshot efektifnya per benefit.
type Resolver struct {
	subscriptions subrepo.SubscriptionRepository
	snapshots     repository.SnapshotReader
}

func NewResolver(subscriptions subrepo.SubscriptionRepository, snapshots repository.SnapshotReader) *Resolver {
	return &Resolver{subscriptions: subscriptions, snapshots: snapshots}
}

// Source adalah hasil Resolver. Subscription nil = tenant tidak berlangganan (semua benefit tidak granted).
type Source struct {
	Subscription *subentity.Subscription
	ByBenefit    map[string][]entity.SnapshotValue
}

func (r *Resolver) Resolve(ctx context.Context, tenantID, solutionID string) (*Source, error) {
	src := &Source{ByBenefit: map[string][]entity.SnapshotValue{}}
	sub, err := r.subscriptions.FindUsable(ctx, tenantID, solutionID)
	if err != nil {
		var de *domainerr.DomainError
		if errors.As(err, &de) && de.Code == subconstant.CodeSubscriptionNotFound {
			return src, nil
		}
		return nil, err
	}
	src.Subscription = sub

	values, err := r.snapshots.EffectiveSnapshots(ctx, sub.ID)
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		src.ByBenefit[v.BenefitID] = append(src.ByBenefit[v.BenefitID], v)
	}
	return src, nil
}
