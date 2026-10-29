// Package dashboard berisi usecase ringkasan lintas domain untuk dashboard console.
package dashboard

import (
	"context"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
)

type DashboardDeps struct {
	Query port.DashboardQueryReadModel
}

type DashboardUseCase struct {
	deps DashboardDeps
}

func NewDashboardUseCase(deps DashboardDeps) *DashboardUseCase {
	return &DashboardUseCase{deps: deps}
}

// AdminSummary menampilkan jumlah user, company, solution, dan client aktif.
//
// Required — auth: user (platform admin) | scope: -
func (uc *DashboardUseCase) AdminSummary(ctx context.Context) (*dto.AdminSummaryResponse, error) {
	s, err := uc.deps.Query.AdminSummary(ctx)
	if err != nil {
		return nil, apperror.Internal(string(apperror.CodeInternal), err)
	}
	return s, nil
}
