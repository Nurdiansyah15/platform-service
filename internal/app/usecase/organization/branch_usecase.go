package organization

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	authzconstant "platform-service/internal/domain/authz/constant"
	authzservice "platform-service/internal/domain/authz/service"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/organization/constant"
	"platform-service/internal/domain/organization/entity"
	"platform-service/internal/domain/organization/event"
	"platform-service/internal/domain/organization/service"
)

type BranchUseCase struct {
	deps   OrganizationDeps
	access *service.AccessPolicy
	now    func() time.Time
}

func NewBranchUseCase(deps OrganizationDeps) *BranchUseCase {
	return &BranchUseCase{deps: deps, access: service.NewAccessPolicy(deps.Members, deps.Subscriptions), now: time.Now}
}

// Create membuat branch di company. Company yang tidak boleh diakses dilaporkan 403/404
// sesuai AccessPolicy (tidak membocorkan data lintas company).
//
// Required — auth: user (platform admin, atau member active dengan cid = companyId) | scope: -
func (uc *BranchUseCase) Create(ctx context.Context, actor *principal.Principal, companyID string, req dto.CreateOrgUnitRequest) (*dto.BranchResponse, error) {
	company, err := accessibleCompany(ctx, uc.access, uc.deps.Companies, actor, companyID)
	if err != nil {
		return nil, err
	}
	if err := ensurePermission(ctx, uc.deps.Admin, actor, companyID, authzconstant.PermBranchManage, authzservice.CompanyScope(companyID)); err != nil {
		return nil, err
	}
	b, err := entity.NewBranch(company.ID, entity.OrgUnitParams{Name: req.Name, Code: req.Code, Status: req.Status}, uc.now())
	if err != nil {
		return nil, mapCompanyDomainError(err)
	}

	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Branches.Save(ctx, b); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx, port.OutboxEvent{
			EventType: event.TypeBranchCreated,
			CompanyID: &company.ID,
			Payload: event.BranchPayload{
				ID: b.ID, CompanyID: b.CompanyID, Code: b.Code.Value(), Name: b.Name,
				Status: b.Status.String(), UpdatedAt: b.UpdatedAt,
			},
		})
	})
	if err != nil {
		return nil, mapCompanyDomainError(err)
	}
	resp := toBranchResponse(b)
	return &resp, nil
}

// List menampilkan branch (belum dihapus) milik company.
//
// Required — auth: user (platform admin, atau member active dengan cid = companyId) | scope: -
func (uc *BranchUseCase) List(ctx context.Context, actor *principal.Principal, companyID string, req dto.ListOrgUnitQuery) ([]dto.BranchResponse, dto.Meta, error) {
	company, err := accessibleCompany(ctx, uc.access, uc.deps.Companies, actor, companyID)
	if err != nil {
		return nil, dto.Meta{}, err
	}
	items, meta, err := uc.deps.BranchQuery.ListBranches(ctx, port.OrgUnitListReadQuery{
		ParentID: company.ID, Status: req.Status, Search: req.Search, PaginationParams: req.PaginationParams,
	})
	if err != nil {
		return nil, dto.Meta{}, mapCompanyDomainError(err)
	}
	out := make([]dto.BranchResponse, 0, len(items))
	for _, it := range items {
		out = append(out, dto.BranchResponse{
			ID: it.ID, CompanyID: it.ParentID, Name: it.Name, Code: it.Code, Status: it.Status, CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt,
		})
	}
	return out, meta, nil
}

// Update mengubah nama / kode / status branch. Branch milik company yang tidak bisa diakses
// dilaporkan 404.
//
// Required — auth: user (platform admin, atau member dengan permission branch.manage di branch / company-nya) atau service | scope: org:write
func (uc *BranchUseCase) Update(ctx context.Context, actor *principal.Principal, branchID string, req dto.UpdateOrgUnitRequest) (*dto.BranchResponse, error) {
	b, err := uc.manageableBranch(ctx, actor, branchID)
	if err != nil {
		return nil, err
	}
	if err := b.Update(req.Name, req.Code, req.Status, uc.now()); err != nil {
		return nil, mapCompanyDomainError(err)
	}
	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Branches.Save(ctx, b); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx, branchEvent(event.TypeBranchUpdated, b))
	})
	if err != nil {
		return nil, mapCompanyDomainError(err)
	}
	resp := toBranchResponse(b)
	return &resp, nil
}

// Delete menghapus branch (soft delete). Role assignment ber-scope branch ini ikut dihapus.
//
// Required — auth: user (platform admin, atau member dengan permission branch.manage di branch / company-nya) atau service | scope: org:write
func (uc *BranchUseCase) Delete(ctx context.Context, actor *principal.Principal, branchID string) error {
	b, err := uc.manageableBranch(ctx, actor, branchID)
	if err != nil {
		return err
	}
	b.Delete(uc.now())
	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Branches.Save(ctx, b); err != nil {
			return err
		}
		if err := uc.deps.RoleAssignments.DeleteByScope(ctx, branchScopeType, b.ID); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx, branchEvent(event.TypeBranchDeleted, b))
	})
	if err != nil {
		return mapCompanyDomainError(err)
	}
	return nil
}

// branchScopeType — scope_type role assignment untuk branch (authz valueobject.ScopeBranch).
const branchScopeType = "branch"

// manageableBranch — accessibleBranch + permission platform branch.manage pada branch itu
// (assignment level company induknya ikut berlaku).
func (uc *BranchUseCase) manageableBranch(ctx context.Context, actor *principal.Principal, branchID string) (*entity.Branch, error) {
	b, err := uc.accessibleBranch(ctx, actor, branchID)
	if err != nil {
		return nil, err
	}
	if err := ensurePermission(ctx, uc.deps.Admin, actor, b.CompanyID, authzconstant.PermBranchManage, authzservice.BranchScope(b.CompanyID, b.ID)); err != nil {
		return nil, err
	}
	return b, nil
}

// accessibleBranch memuat branch lalu memeriksa akses ke company pemiliknya; ditolak → 404.
func (uc *BranchUseCase) accessibleBranch(ctx context.Context, actor *principal.Principal, branchID string) (*entity.Branch, error) {
	b, err := uc.deps.Branches.FindByID(ctx, branchID)
	if err != nil {
		return nil, mapCompanyDomainError(err)
	}
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), b.CompanyID); err != nil {
		if isDomainCode(err, constant.CodeCompanyAccessDenied) {
			return nil, mapCompanyDomainError(domainerr.New(constant.CodeBranchNotFound))
		}
		return nil, mapCompanyDomainError(err)
	}
	return b, nil
}

func branchEvent(eventType string, b *entity.Branch) port.OutboxEvent {
	return port.OutboxEvent{
		EventType: eventType,
		CompanyID: &b.CompanyID,
		Payload: event.BranchPayload{
			ID: b.ID, CompanyID: b.CompanyID, Code: b.Code.Value(), Name: b.Name,
			Status: b.Status.String(), UpdatedAt: b.UpdatedAt,
		},
		OccurredAt: b.UpdatedAt,
	}
}
