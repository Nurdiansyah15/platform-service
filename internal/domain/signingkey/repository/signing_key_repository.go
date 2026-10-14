package repository

import (
	"context"

	"platform-service/internal/domain/signingkey/entity"
)

type SigningKeyRepository interface {
	// Save insert/update key. Insert key active kedua → CodeSigningKeyActiveExists.
	Save(ctx context.Context, k *entity.SigningKey) error
	FindActive(ctx context.Context) (*entity.SigningKey, error)
	FindByKID(ctx context.Context, kid string) (*entity.SigningKey, error)
	// ListPublished mengembalikan key active + retiring (isi JWKS), terbaru dulu.
	ListPublished(ctx context.Context) ([]*entity.SigningKey, error)
}
