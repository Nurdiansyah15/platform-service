package repository

import (
	"context"

	"platform-service/internal/domain/user/entity"
)

type UserRepository interface {
	// Save insert/update. Email bentrok → CodeUserEmailDuplicate.
	Save(ctx context.Context, u *entity.User) error
	FindByID(ctx context.Context, id string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
}
