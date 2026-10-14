package organization

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	"platform-service/internal/domain/organization/entity"
	"platform-service/internal/domain/organization/event"
	"platform-service/internal/domain/tenant/service"
)

type BranchUseCase struct {
	deps   OrganizationDeps
	access *service.AccessPolicy
	now    func() time.Time
}

func NewBranchUseCase(deps OrganizationDeps) *BranchUseCase {
	return &BranchUseCase{deps: deps, access: service.NewAccessPolicy(deps.Members, deps.Subscriptions), now: time.Now}
}

// Create membuat branch di company. Company tenant lain diperlakukan sebagai tidak ada (404).
//
// Required — auth: user (platform admin, atau member active dengan tid = tenant pemilik company) | scope: -
func (uc *BranchUseCase) Create(ctx context.Context, actor *principal.Principal, companyID string, req dto.CreateOrgUnitRequest) (*dto.BranchResponse, error) {
	company, err := accessibleCompany(ctx, uc.access, uc.deps.Companies, actor, companyID)
	if err != nil {
		return nil, err
	}
	b, err := entity.NewBranch(company.ID, entity.OrgUnitParams{Name: req.Name, Code: req.Code, Status: req.Status}, uc.now())
	if err != nil {
		return nil, mapOrgDomainError(err)
	}

	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Branches.Save(ctx, b); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx, port.OutboxEvent{
			EventType: event.TypeBranchCreated,
			TenantID:  &company.TenantID,
			Payload: event.BranchPayload{
				ID: b.ID, TenantID: company.TenantID, CompanyID: b.CompanyID, Code: b.Code.Value(), Name: b.Name,
				Status: b.Status.String(), UpdatedAt: b.UpdatedAt,
			},
		})
	})
	if err != nil {
		return nil, mapOrgDomainError(err)
	}
	resp := toBranchResponse(b)
	return &resp, nil
}

// List menampilkan branch (belum dihapus) milik company.
//
// Required — auth: user (platform admin, atau member active dengan tid = tenant pemilik company) | scope: -
func (uc *BranchUseCase) List(ctx context.Context, actor *principal.Principal, companyID string, req dto.ListOrgUnitQuery) ([]dto.BranchResponse, dto.Meta, error) {
	company, err := accessibleCompany(ctx, uc.access, uc.deps.Companies, actor, companyID)
	if err != nil {
		return nil, dto.Meta{}, err
	}
	items, meta, err := uc.deps.Query.ListBranches(ctx, port.OrgUnitListReadQuery{
		ParentID: company.ID, Status: req.Status, Search: req.Search, PaginationParams: req.PaginationParams,
	})
	if err != nil {
		return nil, dto.Meta{}, mapOrgDomainError(err)
	}
	out := make([]dto.BranchResponse, 0, len(items))
	for _, it := range items {
		out = append(out, dto.BranchResponse{
			ID: it.ID, CompanyID: it.ParentID, Name: it.Name, Code: it.Code, Status: it.Status, CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt,
		})
	}
	return out, meta, nil
}
