package addon

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	"platform-service/internal/domain/addon/constant"
	"platform-service/internal/domain/addon/entity"
	addonrepo "platform-service/internal/domain/addon/repository"
	benefitrepo "platform-service/internal/domain/benefit/repository"
	benefitservice "platform-service/internal/domain/benefit/service"
	domainerr "platform-service/internal/domain/errors"
	pricingvo "platform-service/internal/domain/pricing/valueobject"
	solutionconstant "platform-service/internal/domain/solution/constant"
	solutionrepo "platform-service/internal/domain/solution/repository"
)

type AddonDeps struct {
	Tx            port.Transactor
	Addons        addonrepo.AddonRepository
	AddonBenefits addonrepo.AddonBenefitRepository
	Benefits      benefitrepo.BenefitRepository
	Solutions     solutionrepo.SolutionRepository
	Query         port.AddonQueryReadModel
}

type AddonUseCase struct {
	deps      AddonDeps
	validator *benefitservice.AssignmentValidator
	now       func() time.Time
}

func NewAddonUseCase(deps AddonDeps) *AddonUseCase {
	return &AddonUseCase{deps: deps, validator: benefitservice.NewAssignmentValidator(deps.Benefits), now: time.Now}
}

// Create menambah add-on pada solution.
//
// Required — auth: user (platform admin) | scope: - | deployment: cloud (LICENSE_MANAGED di on_prem)
func (uc *AddonUseCase) Create(ctx context.Context, solutionID string, req dto.CreateAddonRequest) (*dto.AddonResponse, error) {
	if _, err := uc.deps.Solutions.FindByID(ctx, solutionID); err != nil {
		return nil, mapAddonDomainError(err)
	}
	price, err := pricingvo.ParsePriceJSON(req.Price)
	if err != nil {
		return nil, mapAddonDomainError(err)
	}
	a, err := entity.NewAddon(entity.NewAddonParams{
		SolutionID:   solutionID,
		Code:         req.Code,
		Name:         req.Name,
		Description:  req.Description,
		Price:        price,
		Currency:     req.Currency,
		BillingCycle: req.BillingCycle,
		IsActive:     req.IsActive,
	}, uc.now())
	if err != nil {
		return nil, mapAddonDomainError(err)
	}
	if err := uc.deps.Addons.Save(ctx, a); err != nil {
		return nil, mapAddonDomainError(err)
	}
	resp := toAddonResponse(a)
	return &resp, nil
}

// List menampilkan add-on solution. Non-admin hanya melihat add-on aktif dari solution aktif.
//
// Required — auth: user / service (token valid apa pun) | scope: -
func (uc *AddonUseCase) List(ctx context.Context, actor *principal.Principal, solutionID string, req dto.ListAddonQuery) ([]dto.AddonResponse, dto.Meta, error) {
	if err := uc.ensureSolutionVisible(ctx, actor, solutionID); err != nil {
		return nil, dto.Meta{}, err
	}
	isActive := req.IsActive
	if !isPlatformAdmin(actor) {
		active := true
		isActive = &active
	}
	items, meta, err := uc.deps.Query.ListAddons(ctx, port.AddonListReadQuery{
		SolutionID: solutionID, IsActive: isActive, BillingCycle: req.BillingCycle, Search: req.Search,
		PaginationParams: req.PaginationParams,
	})
	if err != nil {
		return nil, dto.Meta{}, mapAddonDomainError(err)
	}
	out := make([]dto.AddonResponse, 0, len(items))
	for _, it := range items {
		out = append(out, dto.AddonResponse{
			ID: it.ID, SolutionID: it.SolutionID, Code: it.Code, Name: it.Name, Description: it.Description,
			Price: it.Price, Currency: it.Currency, BillingCycle: it.BillingCycle, IsActive: it.IsActive,
			CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt,
		})
	}
	return out, meta, nil
}

// ListBenefits menampilkan nilai benefit pada add-on.
//
// Required — auth: user / service (token valid apa pun) | scope: -
func (uc *AddonUseCase) ListBenefits(ctx context.Context, actor *principal.Principal, addonID string) ([]dto.AssignedBenefitResponse, error) {
	a, err := uc.deps.Addons.FindByID(ctx, addonID)
	if err != nil {
		return nil, mapAddonDomainError(err)
	}
	if !isPlatformAdmin(actor) {
		if !a.IsActive || uc.ensureSolutionVisible(ctx, actor, a.SolutionID) != nil {
			return nil, mapAddonDomainError(domainerr.New(constant.CodeAddonNotFound))
		}
	}
	items, err := uc.deps.Query.ListAddonBenefits(ctx, a.ID)
	if err != nil {
		return nil, mapAddonDomainError(err)
	}
	return toAssignedBenefitResponses(items), nil
}

// ReplaceBenefits mengganti seluruh benefit add-on (replace-all, idempotent). Tidak
// memengaruhi subscription berjalan karena entitlement dihitung dari snapshot.
//
// Required — auth: user (platform admin) | scope: - | deployment: cloud (LICENSE_MANAGED di on_prem)
func (uc *AddonUseCase) ReplaceBenefits(ctx context.Context, addonID string, req dto.ReplaceBenefitsRequest) ([]dto.AssignedBenefitResponse, error) {
	a, err := uc.deps.Addons.FindByID(ctx, addonID)
	if err != nil {
		return nil, mapAddonDomainError(err)
	}

	assignments := make([]benefitservice.Assignment, 0, len(req.Benefits))
	for _, b := range req.Benefits {
		assignments = append(assignments, benefitservice.Assignment{BenefitID: b.BenefitID, Value: b.Value})
	}
	validated, err := uc.validator.Validate(ctx, a.SolutionID, assignments)
	if err != nil {
		return nil, mapAddonDomainError(err)
	}

	rows := make([]entity.AddonBenefit, 0, len(validated))
	for _, v := range validated {
		rows = append(rows, entity.AddonBenefit{AddonID: a.ID, BenefitID: v.Benefit.ID, SolutionID: a.SolutionID, Value: v.Value})
	}
	if err := uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		return uc.deps.AddonBenefits.ReplaceAll(ctx, a.ID, rows)
	}); err != nil {
		return nil, mapAddonDomainError(err)
	}

	items, err := uc.deps.Query.ListAddonBenefits(ctx, a.ID)
	if err != nil {
		return nil, mapAddonDomainError(err)
	}
	return toAssignedBenefitResponses(items), nil
}

// ensureSolutionVisible — solution harus ada; solution inactive disembunyikan dari non-admin.
func (uc *AddonUseCase) ensureSolutionVisible(ctx context.Context, actor *principal.Principal, solutionID string) error {
	sol, err := uc.deps.Solutions.FindByID(ctx, solutionID)
	if err != nil {
		return mapAddonDomainError(err)
	}
	if !sol.IsActive() && !isPlatformAdmin(actor) {
		return mapAddonDomainError(domainerr.New(solutionconstant.CodeSolutionNotFound))
	}
	return nil
}
