package user

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/domain/user/constant"
	"platform-service/internal/domain/user/entity"
	userrepo "platform-service/internal/domain/user/repository"
)

type UserDeps struct {
	Repo   userrepo.UserRepository
	Hasher port.PasswordHasher
}

type UserUseCase struct {
	deps UserDeps
	now  func() time.Time
}

func NewUserUseCase(deps UserDeps) *UserUseCase {
	return &UserUseCase{deps: deps, now: time.Now}
}

// Create membuat user baru.
//
// Required — auth: user (platform admin) | scope: -
func (uc *UserUseCase) Create(ctx context.Context, req dto.CreateUserRequest) (*dto.UserResponse, error) {
	hash, err := uc.hashOptional(req.Password)
	if err != nil {
		return nil, mapUserDomainError(err)
	}
	u, err := entity.NewUser(entity.NewUserParams{
		Email:           req.Email,
		Name:            req.Name,
		PasswordHash:    hash,
		Status:          req.Status,
		IsPlatformAdmin: req.IsPlatformAdmin,
	}, uc.now())
	if err != nil {
		return nil, mapUserDomainError(err)
	}
	if err := uc.deps.Repo.Save(ctx, u); err != nil {
		return nil, mapUserDomainError(err)
	}
	resp := ToUserResponse(u)
	return &resp, nil
}

// EnsurePlatformAdmin memastikan user dengan email tersebut ada dan menjadi
// platform admin. Idempotent: user yang sudah ada hanya diberi flag admin —
// password-nya tidak diubah. Dipakai seeder.
//
// Required — auth: internal (seeder) | scope: -
func (uc *UserUseCase) EnsurePlatformAdmin(ctx context.Context, email, name, password string) (*dto.UserResponse, error) {
	existing, err := uc.deps.Repo.FindByEmail(ctx, email)
	switch {
	case err == nil:
		if !existing.IsPlatformAdmin {
			existing.GrantPlatformAdmin(uc.now())
			if err := uc.deps.Repo.Save(ctx, existing); err != nil {
				return nil, mapUserDomainError(err)
			}
		}
		resp := ToUserResponse(existing)
		return &resp, nil
	case !isDomainCode(err, constant.CodeUserNotFound):
		return nil, mapUserDomainError(err)
	}

	return uc.Create(ctx, dto.CreateUserRequest{
		Email:           email,
		Name:            name,
		Password:        &password,
		IsPlatformAdmin: true,
	})
}

func (uc *UserUseCase) hashOptional(password *string) (*string, error) {
	if password == nil {
		return nil, nil
	}
	if err := entity.ValidatePassword(*password); err != nil {
		return nil, err
	}
	h, err := uc.deps.Hasher.Hash(*password)
	if err != nil {
		return nil, err
	}
	return &h, nil
}
