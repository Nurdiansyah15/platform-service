package repository

import (
	"context"

	"platform-service/internal/domain/authsession/entity"
)

type AuthorizationCodeRepository interface {
	Create(ctx context.Context, c *entity.AuthorizationCode) error
	// FindByCodeHash → CodeAuthCodeNotFound bila tidak ada.
	FindByCodeHash(ctx context.Context, hash string) (*entity.AuthorizationCode, error)
	// MarkUsed menyimpan used_at hanya bila kode belum pernah dipakai (compare-and-swap).
	// Kalah balapan → CodeAuthCodeAlreadyUsed.
	MarkUsed(ctx context.Context, c *entity.AuthorizationCode) error
}
