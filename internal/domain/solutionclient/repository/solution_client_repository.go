package repository

import (
	"context"
	"time"

	"platform-service/internal/domain/solutionclient/entity"
)

type SolutionClientRepository interface {
	Save(ctx context.Context, c *entity.SolutionClient) error
	FindByID(ctx context.Context, id string) (*entity.SolutionClient, error)
	FindByClientID(ctx context.Context, clientID string) (*entity.SolutionClient, error)
	// TouchLastUsed mencatat waktu terakhir client menukar token (best-effort).
	TouchLastUsed(ctx context.Context, id string, at time.Time) error
}
