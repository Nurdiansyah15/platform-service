package plan

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	benefitrepo "platform-service/internal/domain/benefit/repository"
	benefitservice "platform-service/internal/domain/benefit/service"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/plan/constant"
	"platform-service/internal/domain/plan/entity"
	planrepo "platform-service/internal/domain/plan/repository"
	pricingvo "platform-service/internal/domain/pricing/valueobject"
	solutionconstant "platform-service/internal/domain/solution/constant"
	solutionrepo "platform-service/internal/domain/solution/repository"
)

type PlanDeps struct {
	Tx           port.Transactor
	Plans        planrepo.PlanRepository
	PlanBenefits planrepo.PlanBenefitRepository
	Benefits     benefitrepo.BenefitRepository
	Solutions    solutionrepo.SolutionRepository
	Query        port.PlanQueryReadModel
}

type PlanUseCase struct {
	deps      PlanDeps
	validator *benefitservice.AssignmentValidator
	now       func() time.Time
}

func NewPlanUseCase(deps PlanDeps) *PlanUseCase {
	return &PlanUseCase{deps: deps, validator: benefitservice.NewAssignmentValidator(deps.Benefits), now: time.Now}
}

// Create menambah plan pada solution.
//
// Required — auth: user (platform admin) | scope: - | deployment: cloud (LICENSE_MANAGED di on_prem)
func (uc *PlanUseCase) Create(ctx context.Context, solutionID string, req dto.CreatePlanRequest) (*dto.PlanResponse, error) {
	if _, err := uc.deps.Solutions.FindByID(ctx, solutionID); err != nil {
		return nil, mapPlanDomainError(err)
	}
	price, err := pricingvo.ParsePriceJSON(req.Price)
	if err != nil {
		return nil, mapPlanDomainError(err)
	}
	p, err := entity.NewPlan(entity.NewPlanParams{
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
		return nil, mapPlanDomainError(err)
	}
	if err := uc.deps.Plans.Save(ctx, p); err != nil {
		return nil, mapPlanDomainError(err)
	}
	resp := toPlanResponse(p)
	return &resp, nil
}

// List menampilkan plan solution. Non-admin hanya melihat plan aktif dari solution aktif.
//
// Required — auth: user / service (token valid apa pun) | scope: -
func (uc *PlanUseCase) List(ctx context.Context, actor *principal.Principal, solutionID string, req dto.ListPlanQuery) ([]dto.PlanResponse, dto.Meta, error) {
	if err := uc.ensureSolutionVisible(ctx, actor, solutionID); err != nil {
		return nil, dto.Meta{}, err
	}
	isActive := req.IsActive
	if !isPlatformAdmin(actor) {
		active := true
		isActive = &active
	}
	items, meta, err := uc.deps.Query.ListPlans(ctx, port.PlanListReadQuery{
		SolutionID: solutionID, IsActive: isActive, BillingCycle: req.BillingCycle, Search: req.Search,
		PaginationParams: req.PaginationParams,
	})
	if err != nil {
		return nil, dto.Meta{}, mapPlanDomainError(err)
	}
	out := make([]dto.PlanResponse, 0, len(items))
	for _, it := range items {
		out = append(out, dto.PlanResponse{
			ID: it.ID, SolutionID: it.SolutionID, Code: it.Code, Name: it.Name, Description: it.Description,
			Price: it.Price, Currency: it.Currency, BillingCycle: it.BillingCycle, IsActive: it.IsActive,
			CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt,
		})
	}
	return out, meta, nil
}

// ListBenefits menampilkan nilai benefit pada plan.
//
// Required — auth: user / service (token valid apa pun) | scope: -
func (uc *PlanUseCase) ListBenefits(ctx context.Context, actor *principal.Principal, planID string) ([]dto.AssignedBenefitResponse, error) {
	p, err := uc.deps.Plans.FindByID(ctx, planID)
	if err != nil {
		return nil, mapPlanDomainError(err)
	}
	if !isPlatformAdmin(actor) {
		if !p.IsActive || uc.ensureSolutionVisible(ctx, actor, p.SolutionID) != nil {
			return nil, mapPlanDomainError(domainerr.New(constant.CodePlanNotFound))
		}
	}
	items, err := uc.deps.Query.ListPlanBenefits(ctx, p.ID)
	if err != nil {
		return nil, mapPlanDomainError(err)
	}
	return toAssignedBenefitResponses(items), nil
}

// ReplaceBenefits mengganti seluruh benefit plan (replace-all, idempotent). Tidak
// memengaruhi subscription berjalan karena entitlement dihitung dari snapshot.
//
// Required — auth: user (platform admin) | scope: - | deployment: cloud (LICENSE_MANAGED di on_prem)
func (uc *PlanUseCase) ReplaceBenefits(ctx context.Context, planID string, req dto.ReplaceBenefitsRequest) ([]dto.AssignedBenefitResponse, error) {
	p, err := uc.deps.Plans.FindByID(ctx, planID)
	if err != nil {
		return nil, mapPlanDomainError(err)
	}

	assignments := make([]benefitservice.Assignment, 0, len(req.Benefits))
	for _, b := range req.Benefits {
		assignments = append(assignments, benefitservice.Assignment{BenefitID: b.BenefitID, Value: b.Value})
	}
	validated, err := uc.validator.Validate(ctx, p.SolutionID, assignments)
	if err != nil {
		return nil, mapPlanDomainError(err)
	}

	rows := make([]entity.PlanBenefit, 0, len(validated))
	for _, v := range validated {
		rows = append(rows, entity.PlanBenefit{PlanID: p.ID, BenefitID: v.Benefit.ID, SolutionID: p.SolutionID, Value: v.Value})
	}
	if err := uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		return uc.deps.PlanBenefits.ReplaceAll(ctx, p.ID, rows)
	}); err != nil {
		return nil, mapPlanDomainError(err)
	}

	items, err := uc.deps.Query.ListPlanBenefits(ctx, p.ID)
	if err != nil {
		return nil, mapPlanDomainError(err)
	}
	return toAssignedBenefitResponses(items), nil
}

// ensureSolutionVisible — solution harus ada; solution inactive disembunyikan dari non-admin.
func (uc *PlanUseCase) ensureSolutionVisible(ctx context.Context, actor *principal.Principal, solutionID string) error {
	sol, err := uc.deps.Solutions.FindByID(ctx, solutionID)
	if err != nil {
		return mapPlanDomainError(err)
	}
	if !sol.IsActive() && !isPlatformAdmin(actor) {
		return mapPlanDomainError(domainerr.New(solutionconstant.CodeSolutionNotFound))
	}
	return nil
}
