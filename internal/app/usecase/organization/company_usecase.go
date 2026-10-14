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
	tenantrepo "platform-service/internal/domain/tenant/repository"
	"platform-service/internal/domain/tenant/service"
)

// OrganizationDeps dipakai bersama CompanyUseCase dan BranchUseCase.
type OrganizationDeps struct {
	Tx        port.Transactor
	Outbox    port.EventOutbox
	Tenants   tenantrepo.TenantRepository
	Members   tenantrepo.TenantMemberRepository
	Companies orgrepo.CompanyRepository
	Branches  orgrepo.BranchRepository
	Query     port.OrganizationQueryReadModel
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

// Create membuat company di tenant.
//
// Required — auth: user (platform admin, atau member active dengan tid = tenantId) | scope: -
func (uc *CompanyUseCase) Create(ctx context.Context, actor *principal.Principal, tenantID string, req dto.CreateOrgUnitRequest) (*dto.CompanyResponse, error) {
	if err := ensureTenant(ctx, uc.access, uc.deps.Tenants, actor, tenantID); err != nil {
		return nil, err
	}
	c, err := entity.NewCompany(tenantID, entity.OrgUnitParams{Name: req.Name, Code: req.Code, Status: req.Status}, uc.now())
	if err != nil {
		return nil, mapOrgDomainError(err)
	}

	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Companies.Save(ctx, c); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx, port.OutboxEvent{
			EventType: event.TypeCompanyCreated,
			TenantID:  &c.TenantID,
			Payload: event.CompanyPayload{
				ID: c.ID, TenantID: c.TenantID, Code: c.Code.Value(), Name: c.Name, Status: c.Status.String(), UpdatedAt: c.UpdatedAt,
			},
		})
	})
	if err != nil {
		return nil, mapOrgDomainError(err)
	}
	resp := toCompanyResponse(c)
	return &resp, nil
}

// List menampilkan company (belum dihapus) milik tenant.
//
// Required — auth: user (platform admin, atau member active dengan tid = tenantId) | scope: -
func (uc *CompanyUseCase) List(ctx context.Context, actor *principal.Principal, tenantID string, req dto.ListOrgUnitQuery) ([]dto.CompanyResponse, dto.Meta, error) {
	if err := ensureTenant(ctx, uc.access, uc.deps.Tenants, actor, tenantID); err != nil {
		return nil, dto.Meta{}, err
	}
	items, meta, err := uc.deps.Query.ListCompanies(ctx, port.OrgUnitListReadQuery{
		ParentID: tenantID, Status: req.Status, Search: req.Search, PaginationParams: req.PaginationParams,
	})
	if err != nil {
		return nil, dto.Meta{}, mapOrgDomainError(err)
	}
	out := make([]dto.CompanyResponse, 0, len(items))
	for _, it := range items {
		out = append(out, dto.CompanyResponse{
			ID: it.ID, TenantID: it.ParentID, Name: it.Name, Code: it.Code, Status: it.Status, CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt,
		})
	}
	return out, meta, nil
}
