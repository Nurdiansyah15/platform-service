package port

import (
	"context"

	"platform-service/internal/app/dto"
)

type DashboardQueryReadModel interface {
	AdminSummary(ctx context.Context) (*dto.AdminSummaryResponse, error)
}
