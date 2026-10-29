package organization

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	authzrepo "platform-service/internal/domain/authz/repository"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/organization/constant"
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
	// RoleAssignments dipakai saat member dikeluarkan (assignment-nya ikut dihapus).
	RoleAssignments authzrepo.RoleAssignmentRepository
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

// Update mengubah nama / slug company; status hanya boleh diubah platform admin.
//
// Required — auth: user (platform admin, atau member active dengan cid = companyId) atau service | scope: org:write
func (uc *CompanyUseCase) Update(ctx context.Context, actor *principal.Principal, companyID string, req dto.UpdateCompanyRequest) (*dto.CompanyResponse, error) {
	isPlatformAdmin := actor.IsUser() && actor.IsPlatformAdmin
	if req.Status != nil && !isPlatformAdmin {
		return nil, mapCompanyDomainError(domainerr.New(constant.CodeCompanyStatusForbidden))
	}
	c, err := accessibleCompany(ctx, uc.access, uc.deps.Companies, actor, companyID)
	if err != nil {
		return nil, err
	}

	now := uc.now()
	if err := c.Update(req.Name, req.Slug, now); err != nil {
		return nil, mapCompanyDomainError(err)
	}
	if req.Status != nil {
		if _, err := c.ChangeStatus(*req.Status, now); err != nil {
			return nil, mapCompanyDomainError(err)
		}
	}

	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Companies.Save(ctx, c); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx, companyEvent(event.TypeCompanyUpdated, c))
	})
	if err != nil {
		return nil, mapCompanyDomainError(err)
	}

	resp := toCompanyResponse(c)
	return &resp, nil
}

func companyEvent(eventType string, c *entity.Company) port.OutboxEvent {
	companyID := c.ID
	return port.OutboxEvent{
		EventType: eventType,
		CompanyID: &companyID,
		Payload: event.CompanyPayload{
			ID: c.ID, Slug: c.Slug.Value(), Name: c.Name, Status: c.Status.String(), UpdatedAt: c.UpdatedAt,
		},
		OccurredAt: c.UpdatedAt,
	}
}

func memberAddedEvent(m *entity.CompanyMember) port.OutboxEvent {
	return memberEvent(event.TypeCompanyMemberAdded, m, m.JoinedAt)
}

func memberEvent(eventType string, m *entity.CompanyMember, at time.Time) port.OutboxEvent {
	companyID := m.CompanyID
	return port.OutboxEvent{
		EventType: eventType,
		CompanyID: &companyID,
		Payload: event.CompanyMemberPayload{
			CompanyID: m.CompanyID, UserID: m.UserID, Status: m.Status.String(), JoinedAt: m.JoinedAt,
		},
		OccurredAt: at,
	}
}
