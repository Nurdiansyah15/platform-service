package organization

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	"platform-service/internal/domain/organization/entity"
	"platform-service/internal/domain/organization/event"
	orgrepo "platform-service/internal/domain/organization/repository"
	"platform-service/internal/domain/organization/service"
	userrepo "platform-service/internal/domain/user/repository"
)

// OrganizationDeps dipakai bersama CompanyUseCase, CompanyMemberUseCase, dan BranchUseCase.
type OrganizationDeps struct {
	Tx           port.Transactor
	Outbox       port.EventOutbox
	Companies    orgrepo.CompanyRepository
	Members      orgrepo.CompanyMemberRepository
	Branches     orgrepo.BranchRepository
	Users        userrepo.UserRepository
	CompanyQuery port.CompanyQueryReadModel
	BranchQuery  port.OrganizationQueryReadModel
	// Subscriptions dipakai AccessPolicy untuk akses token service.
	Subscriptions service.SubscriptionChecker
}

type CompanyUseCase struct {
	deps   OrganizationDeps
	access *service.AccessPolicy
	now    func() time.Time
}

func NewCompanyUseCase(deps OrganizationDeps) *CompanyUseCase {
	return &CompanyUseCase{deps: deps, access: service.NewAccessPolicy(deps.Members, deps.Subscriptions), now: time.Now}
}

// Create membuat company baru. Pembuat non-admin otomatis menjadi member active;
// platform admin tidak (admin membuat company untuk pelanggan lalu menambah member).
//
// Required — auth: user | scope: -
func (uc *CompanyUseCase) Create(ctx context.Context, actor *principal.Principal, req dto.CreateCompanyRequest) (*dto.CompanyResponse, error) {
	now := uc.now()
	c, err := entity.NewCompany(req.Name, req.Slug, now)
	if err != nil {
		return nil, mapCompanyDomainError(err)
	}

	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Companies.Save(ctx, c); err != nil {
			return err
		}
		if actor.IsPlatformAdmin {
			return nil
		}
		m, err := entity.NewCompanyMember(c.ID, actor.UserID, "", now)
		if err != nil {
			return err
		}
		if err := uc.deps.Members.Save(ctx, m); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx, memberAddedEvent(m))
	})
	if err != nil {
		return nil, mapCompanyDomainError(err)
	}

	resp := toCompanyResponse(c)
	return &resp, nil
}

// List menampilkan company: platform admin melihat semua, user lain hanya company
// tempat ia member active. Endpoint pemilihan company — boleh dipanggil tanpa cid.
//
// Required — auth: user | scope: -
func (uc *CompanyUseCase) List(ctx context.Context, actor *principal.Principal, req dto.ListCompanyQuery) ([]dto.CompanyResponse, dto.Meta, error) {
	q := port.CompanyListReadQuery{Status: req.Status, Search: req.Search, PaginationParams: req.PaginationParams}
	if !actor.IsPlatformAdmin {
		q.MemberUserID = &actor.UserID
	}
	items, meta, err := uc.deps.CompanyQuery.ListCompanies(ctx, q)
	if err != nil {
		return nil, dto.Meta{}, mapCompanyDomainError(err)
	}
	out := make([]dto.CompanyResponse, 0, len(items))
	for _, it := range items {
		out = append(out, dto.CompanyResponse{
			ID: it.ID, Name: it.Name, Slug: it.Slug, Status: it.Status, CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt,
		})
	}
	return out, meta, nil
}

// Get mengambil detail company.
//
// Required — auth: user (platform admin, atau member active dengan cid = companyId) | scope: -
func (uc *CompanyUseCase) Get(ctx context.Context, actor *principal.Principal, companyID string) (*dto.CompanyResponse, error) {
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), companyID); err != nil {
		return nil, mapCompanyDomainError(err)
	}
	c, err := uc.deps.Companies.FindByID(ctx, companyID)
	if err != nil {
		return nil, mapCompanyDomainError(err)
	}
	resp := toCompanyResponse(c)
	return &resp, nil
}

func memberAddedEvent(m *entity.CompanyMember) port.OutboxEvent {
	companyID := m.CompanyID
	return port.OutboxEvent{
		EventType: event.TypeCompanyMemberAdded,
		CompanyID: &companyID,
		Payload: event.CompanyMemberPayload{
			CompanyID: m.CompanyID, UserID: m.UserID, Status: m.Status.String(), JoinedAt: m.JoinedAt,
		},
		OccurredAt: m.JoinedAt,
	}
}
