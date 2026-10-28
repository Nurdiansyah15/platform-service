package repository

import (
	"context"

	"platform-service/internal/domain/authsession/entity"
)

type AuthSessionRepository interface {
	Create(ctx context.Context, s *entity.AuthSession) error
	FindByID(ctx context.Context, id string) (*entity.AuthSession, error)
	FindByRefreshTokenHash(ctx context.Context, hash string) (*entity.AuthSession, error)
	// SaveRotation menyimpan hasil Rotate / SwitchCompany (hash, expiry, last_used_at,
	// company_id) hanya bila refresh_token_hash di DB masih previousHash
	// (compare-and-swap). Kalah balapan → CodeSessionRotationConflict.
	SaveRotation(ctx context.Context, s *entity.AuthSession, previousHash string) error
	SaveRevocation(ctx context.Context, s *entity.AuthSession) error
}
