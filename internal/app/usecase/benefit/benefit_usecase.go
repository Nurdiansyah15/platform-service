package benefit

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	"platform-service/internal/domain/benefit/constant"
	"platform-service/internal/domain/benefit/entity"
	benefitrepo "platform-service/internal/domain/benefit/repository"
	domainerr "platform-service/internal/domain/errors"
	solutionconstant "platform-service/internal/domain/solution/constant"
	solutionrepo "platform-service/internal/domain/solution/repository"
)

type BenefitDeps struct {
	Repo      benefitrepo.BenefitRepository
	Query     port.BenefitQueryReadModel
	Solutions solutionrepo.SolutionRepository
}

type BenefitUseCase struct {
	deps BenefitDeps
	now  func() time.Time
}

func NewBenefitUseCase(deps BenefitDeps) *BenefitUseCase {
	return &BenefitUseCase{deps: deps, now: time.Now}
}

// Create menambah master benefit pada solution.
//
// Required — auth: user (platform admin) | scope: - | deployment: cloud (LICENSE_MANAGED di on_prem)
func (uc *BenefitUseCase) Create(ctx context.Context, solutionID string, req dto.CreateBenefitRequest) (*dto.BenefitResponse, error) {
	if _, err := uc.deps.Solutions.FindByID(ctx, solutionID); err != nil {
		return nil, mapBenefitDomainError(err)
	}
	b, err := entity.NewBenefit(entity.NewBenefitParams{
		SolutionID:          solutionID,
		Code:                req.Code,
		Name:                req.Name,
		Description:         req.Description,
		ValueType:           req.ValueType,
		AggregationStrategy: req.AggregationStrategy,
		Unit:                req.Unit,
		Metering:            req.Metering,
		UsageReset:          req.UsageReset,
	}, uc.now())
	if err != nil {
		return nil, mapBenefitDomainError(err)
	}
	if err := uc.deps.Repo.Save(ctx, b); err != nil {
		return nil, mapBenefitDomainError(err)
	}
	resp := toBenefitResponse(b)
	return &resp, nil
}

// List menampilkan benefit milik solution. Solution inactive hanya terlihat oleh platform admin.
//
// Required — auth: user / service (token valid apa pun) | scope: -
func (uc *BenefitUseCase) List(ctx context.Context, actor *principal.Principal, solutionID string, req dto.ListBenefitQuery) ([]dto.BenefitResponse, dto.Meta, error) {
	if err := uc.ensureSolutionVisible(ctx, actor, solutionID); err != nil {
		return nil, dto.Meta{}, err
	}
	items, meta, err := uc.deps.Query.List(ctx, port.BenefitListReadQuery{
		SolutionID: solutionID, ValueType: req.ValueType, Metering: req.Metering, Search: req.Search,
		PaginationParams: req.PaginationParams,
	})
	if err != nil {
		return nil, dto.Meta{}, mapBenefitDomainError(err)
	}
	return items, meta, nil
}

// Get mengambil detail benefit.
//
// Required — auth: user / service (token valid apa pun) | scope: -
func (uc *BenefitUseCase) Get(ctx context.Context, actor *principal.Principal, benefitID string) (*dto.BenefitResponse, error) {
	b, err := uc.deps.Repo.FindByID(ctx, benefitID)
	if err != nil {
		return nil, mapBenefitDomainError(err)
	}
	if err := uc.ensureSolutionVisible(ctx, actor, b.SolutionID); err != nil {
		// Benefit dari solution yang disembunyikan dilaporkan sebagai benefit tidak ditemukan.
		return nil, mapBenefitDomainError(domainerr.New(constant.CodeBenefitNotFound))
	}
	resp := toBenefitResponse(b)
	return &resp, nil
}

// ensureSolutionVisible — solution harus ada; solution inactive disembunyikan dari non-admin.
func (uc *BenefitUseCase) ensureSolutionVisible(ctx context.Context, actor *principal.Principal, solutionID string) error {
	sol, err := uc.deps.Solutions.FindByID(ctx, solutionID)
	if err != nil {
		return mapBenefitDomainError(err)
	}
	if !sol.IsActive() && !isPlatformAdmin(actor) {
		return mapBenefitDomainError(domainerr.New(solutionconstant.CodeSolutionNotFound))
	}
	return nil
}
