package system

import (
	"context"
	"log/slog"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
)

const healthPingTimeout = 2 * time.Second

type SystemDeps struct {
	DB             port.DatabaseHealth
	DeploymentMode string
	Logger         *slog.Logger
}

type SystemUseCase struct {
	deps SystemDeps
}

func NewSystemUseCase(deps SystemDeps) *SystemUseCase {
	return &SystemUseCase{deps: deps}
}

// Health memeriksa kesiapan dependensi (saat ini: database).
//
// Required — auth: publik | scope: -
func (uc *SystemUseCase) Health(ctx context.Context) dto.HealthResponse {
	resp := dto.HealthResponse{
		Status:         dto.HealthStatusOK,
		Database:       dto.HealthStatusOK,
		DeploymentMode: uc.deps.DeploymentMode,
	}

	pingCtx, cancel := context.WithTimeout(ctx, healthPingTimeout)
	defer cancel()
	if err := uc.deps.DB.Ping(pingCtx); err != nil {
		uc.deps.Logger.WarnContext(ctx, "health: database ping gagal", slog.Any("error", err))
		resp.Status = dto.HealthStatusDegraded
		resp.Database = dto.HealthStatusDegraded
	}
	return resp
}
