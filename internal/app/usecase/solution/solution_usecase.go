package solution

import (
	"context"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	"platform-service/internal/domain/solution/entity"
	solutionrepo "platform-service/internal/domain/solution/repository"
	"platform-service/internal/domain/solution/valueobject"
)

type SolutionDeps struct {
	Repo  solutionrepo.SolutionRepository
	Query port.SolutionQueryReadModel
}

type SolutionUseCase struct {
	deps SolutionDeps
}

func NewSolutionUseCase(deps SolutionDeps) *SolutionUseCase {
	return &SolutionUseCase{deps: deps}
}

// Create mendaftarkan solution baru ke katalog.
//
// Required — auth: user (platform admin) | scope: - | deployment: cloud (LICENSE_MANAGED di on_prem)
func (uc *SolutionUseCase) Create(ctx context.Context, req dto.CreateSolutionRequest) (*dto.SolutionResponse, error) {
	s, err := entity.NewSolution(entity.NewSolutionParams{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
	})
	if err != nil {
		return nil, mapSolutionDomainError(err)
	}
	if err := uc.deps.Repo.Save(ctx, s); err != nil {
		return nil, mapSolutionDomainError(err)
	}
	resp := toSolutionResponse(s)
	return &resp, nil
}

// List menampilkan katalog solution. Selain platform admin hanya melihat solution active.
//
// Required — auth: user / service (token valid apa pun) | scope: -
func (uc *SolutionUseCase) List(ctx context.Context, actor *principal.Principal, req dto.ListSolutionQuery) ([]dto.SolutionResponse, dto.Meta, error) {
	status := req.Status
	if !isPlatformAdmin(actor) {
		active := valueobject.SolutionStatusActive.String()
		status = &active
	}

	items, meta, err := uc.deps.Query.List(ctx, port.SolutionListReadQuery{
		Status:           status,
		Search:           req.Search,
		PaginationParams: req.PaginationParams,
	})
	if err != nil {
		return nil, dto.Meta{}, mapSolutionDomainError(err)
	}

	out := make([]dto.SolutionResponse, 0, len(items))
	for _, it := range items {
		out = append(out, dto.SolutionResponse{
			ID:                       it.ID,
			Code:                     it.Code,
			Name:                     it.Name,
			Description:              it.Description,
			Status:                   it.Status,
			PermissionCatalogVersion: it.PermissionCatalogVersion,
			CreatedAt:                it.CreatedAt,
			UpdatedAt:                it.UpdatedAt,
		})
	}
	return out, meta, nil
}

// GetByID mengambil detail solution. Solution inactive hanya terlihat oleh platform admin.
//
// Required — auth: user / service (token valid apa pun) | scope: -
func (uc *SolutionUseCase) GetByID(ctx context.Context, actor *principal.Principal, id string) (*dto.SolutionResponse, error) {
	s, err := uc.deps.Repo.FindByID(ctx, id)
	if err != nil {
		return nil, mapSolutionDomainError(err)
	}
	if !s.IsActive() && !isPlatformAdmin(actor) {
		return nil, errSolutionNotFound()
	}
	resp := toSolutionResponse(s)
	return &resp, nil
}
