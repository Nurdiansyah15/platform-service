package tenant

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	"platform-service/internal/domain/tenant/entity"
	"platform-service/internal/domain/tenant/event"
	tenantrepo "platform-service/internal/domain/tenant/repository"
	"platform-service/internal/domain/tenant/service"
	userrepo "platform-service/internal/domain/user/repository"
)

// TenantDeps dipakai bersama TenantUseCase dan MemberUseCase.
type TenantDeps struct {
	Tx      port.Transactor
	Outbox  port.EventOutbox
	Tenants tenantrepo.TenantRepository
	Members tenantrepo.TenantMemberRepository
	Users   userrepo.UserRepository
	Query   port.TenantQueryReadModel
	// Subscriptions dipakai AccessPolicy untuk akses token service.
	Subscriptions service.SubscriptionChecker
}

type TenantUseCase struct {
	deps   TenantDeps
	access *service.AccessPolicy
	now    func() time.Time
}

func NewTenantUseCase(deps TenantDeps) *TenantUseCase {
	return &TenantUseCase{deps: deps, access: service.NewAccessPolicy(deps.Members, deps.Subscriptions), now: time.Now}
}

// Create membuat tenant baru. Pembuat non-admin otomatis menjadi member active;
// platform admin tidak (admin membuat tenant untuk pelanggan lalu menambah member).
//
// Required — auth: user | scope: -
func (uc *TenantUseCase) Create(ctx context.Context, actor *principal.Principal, req dto.CreateTenantRequest) (*dto.TenantResponse, error) {
	now := uc.now()
	t, err := entity.NewTenant(req.Name, req.Slug, now)
	if err != nil {
		return nil, mapTenantDomainError(err)
	}

	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Tenants.Save(ctx, t); err != nil {
			return err
		}
		if actor.IsPlatformAdmin {
			return nil
		}
		m, err := entity.NewTenantMember(t.ID, actor.UserID, "", now)
		if err != nil {
			return err
		}
		if err := uc.deps.Members.Save(ctx, m); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx, memberAddedEvent(m))
	})
	if err != nil {
		return nil, mapTenantDomainError(err)
	}

	resp := toTenantResponse(t)
	return &resp, nil
}

// List menampilkan tenant: platform admin melihat semua, user lain hanya tenant
// tempat ia member active. Endpoint pemilihan tenant — boleh dipanggil tanpa tid.
//
// Required — auth: user | scope: -
func (uc *TenantUseCase) List(ctx context.Context, actor *principal.Principal, req dto.ListTenantQuery) ([]dto.TenantResponse, dto.Meta, error) {
	q := port.TenantListReadQuery{Status: req.Status, Search: req.Search, PaginationParams: req.PaginationParams}
	if !actor.IsPlatformAdmin {
		q.MemberUserID = &actor.UserID
	}
	items, meta, err := uc.deps.Query.ListTenants(ctx, q)
	if err != nil {
		return nil, dto.Meta{}, mapTenantDomainError(err)
	}
	out := make([]dto.TenantResponse, 0, len(items))
	for _, it := range items {
		out = append(out, dto.TenantResponse{
			ID: it.ID, Name: it.Name, Slug: it.Slug, Status: it.Status, CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt,
		})
	}
	return out, meta, nil
}

// Get mengambil detail tenant.
//
// Required — auth: user (platform admin, atau member active dengan tid = tenantId) | scope: -
func (uc *TenantUseCase) Get(ctx context.Context, actor *principal.Principal, tenantID string) (*dto.TenantResponse, error) {
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), tenantID); err != nil {
		return nil, mapTenantDomainError(err)
	}
	t, err := uc.deps.Tenants.FindByID(ctx, tenantID)
	if err != nil {
		return nil, mapTenantDomainError(err)
	}
	resp := toTenantResponse(t)
	return &resp, nil
}

func memberAddedEvent(m *entity.TenantMember) port.OutboxEvent {
	tenantID := m.TenantID
	return port.OutboxEvent{
		EventType: event.TypeTenantMemberAdded,
		TenantID:  &tenantID,
		Payload: event.TenantMemberPayload{
			TenantID: m.TenantID, UserID: m.UserID, Status: m.Status.String(), JoinedAt: m.JoinedAt,
		},
		OccurredAt: m.JoinedAt,
	}
}
