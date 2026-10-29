package user

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/user/constant"
	"platform-service/internal/domain/user/entity"
	"platform-service/internal/domain/user/event"
	userrepo "platform-service/internal/domain/user/repository"
	"platform-service/internal/domain/user/valueobject"
)

type UserDeps struct {
	Repo   userrepo.UserRepository
	Hasher port.PasswordHasher
	// Tx, Outbox, dan Query tidak dipakai seeder (EnsurePlatformAdmin).
	Tx     port.Transactor
	Outbox port.EventOutbox
	Query  port.UserQueryReadModel
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

// List menampilkan semua user.
//
// Required — auth: user (platform admin) | scope: -
func (uc *UserUseCase) List(ctx context.Context, req dto.ListUserQuery) ([]dto.UserResponse, dto.Meta, error) {
	items, meta, err := uc.deps.Query.List(ctx, port.UserListReadQuery{
		Status: req.Status, IsPlatformAdmin: req.IsPlatformAdmin, Search: req.Search, PaginationParams: req.PaginationParams,
	})
	if err != nil {
		return nil, dto.Meta{}, mapUserDomainError(err)
	}
	return items, meta, nil
}

// Get mengambil detail user beserta keanggotaan company-nya.
//
// Required — auth: user (platform admin) | scope: -
func (uc *UserUseCase) Get(ctx context.Context, id string) (*dto.UserDetailResponse, error) {
	u, err := uc.deps.Repo.FindByID(ctx, id)
	if err != nil {
		return nil, mapUserDomainError(err)
	}
	memberships, err := uc.deps.Query.ListMemberships(ctx, id)
	if err != nil {
		return nil, mapUserDomainError(err)
	}
	return &dto.UserDetailResponse{UserResponse: ToUserResponse(u), Memberships: memberships}, nil
}

// Update mengubah email, nama, flag platform admin, atau me-reset password user.
// Admin tidak boleh mencabut flag admin dirinya sendiri.
//
// Required — auth: user (platform admin) | scope: -
func (uc *UserUseCase) Update(ctx context.Context, actor *principal.Principal, id string, req dto.UpdateUserRequest) (*dto.UserResponse, error) {
	if id == actor.UserID && req.IsPlatformAdmin != nil && !*req.IsPlatformAdmin {
		return nil, mapUserDomainError(domainerr.New(constant.CodeUserSelfChangeForbidden))
	}
	u, err := uc.deps.Repo.FindByID(ctx, id)
	if err != nil {
		return nil, mapUserDomainError(err)
	}
	now := uc.now()
	if err := u.Update(req.Name, req.Email, now); err != nil {
		return nil, mapUserDomainError(err)
	}
	if req.IsPlatformAdmin != nil {
		u.SetPlatformAdmin(*req.IsPlatformAdmin, now)
	}
	if err := uc.setPassword(u, req.Password, now); err != nil {
		return nil, mapUserDomainError(err)
	}
	return uc.save(ctx, u, event.TypeUserUpdated)
}

// ChangeStatus mengubah status user. User yang tidak active langsung tidak bisa memakai
// token-nya (ResolveUserPrincipal mengecek status di setiap request).
// Admin tidak boleh menonaktifkan dirinya sendiri.
//
// Required — auth: user (platform admin) | scope: -
func (uc *UserUseCase) ChangeStatus(ctx context.Context, actor *principal.Principal, id string, req dto.UpdateUserStatusRequest) (*dto.UserResponse, error) {
	if id == actor.UserID && req.Status != valueobject.UserStatusActive.String() {
		return nil, mapUserDomainError(domainerr.New(constant.CodeUserSelfChangeForbidden))
	}
	u, err := uc.deps.Repo.FindByID(ctx, id)
	if err != nil {
		return nil, mapUserDomainError(err)
	}
	changed, err := u.ChangeStatus(req.Status, uc.now())
	if err != nil {
		return nil, mapUserDomainError(err)
	}
	if !changed {
		resp := ToUserResponse(u)
		return &resp, nil
	}
	return uc.save(ctx, u, event.TypeUserStatusChanged)
}

// Me mengambil data user yang sedang login.
//
// Required — auth: user | scope: -
func (uc *UserUseCase) Me(ctx context.Context, actor *principal.Principal) (*dto.UserResponse, error) {
	u, err := uc.deps.Repo.FindByID(ctx, actor.UserID)
	if err != nil {
		return nil, mapUserDomainError(err)
	}
	resp := ToUserResponse(u)
	return &resp, nil
}

// UpdateMe mengubah nama / password user yang sedang login. Ganti password wajib
// menyertakan password lama.
//
// Required — auth: user | scope: -
func (uc *UserUseCase) UpdateMe(ctx context.Context, actor *principal.Principal, req dto.UpdateMeRequest) (*dto.UserResponse, error) {
	u, err := uc.deps.Repo.FindByID(ctx, actor.UserID)
	if err != nil {
		return nil, mapUserDomainError(err)
	}
	if req.Password != nil {
		// User tanpa password (diundang) belum bisa login, jadi selalu punya password di sini.
		if req.CurrentPassword == nil || !u.HasPassword() || !uc.deps.Hasher.Verify(*u.PasswordHash, *req.CurrentPassword) {
			return nil, mapUserDomainError(domainerr.New(constant.CodeUserCurrentPasswordInvalid))
		}
	}
	now := uc.now()
	if err := u.Update(req.Name, nil, now); err != nil {
		return nil, mapUserDomainError(err)
	}
	if err := uc.setPassword(u, req.Password, now); err != nil {
		return nil, mapUserDomainError(err)
	}
	return uc.save(ctx, u, event.TypeUserUpdated)
}

func (uc *UserUseCase) setPassword(u *entity.User, password *string, now time.Time) error {
	hash, err := uc.hashOptional(password)
	if err != nil || hash == nil {
		return err
	}
	u.SetPassword(*hash, now)
	return nil
}

// save menyimpan user dan menulis event outbox di transaksi yang sama.
func (uc *UserUseCase) save(ctx context.Context, u *entity.User, eventType string) (*dto.UserResponse, error) {
	err := uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Repo.Save(ctx, u); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx, port.OutboxEvent{
			EventType: eventType,
			Payload: event.UserPayload{
				UserID: u.ID, Email: u.Email.Value(), Name: u.Name, Status: u.Status.String(),
				IsPlatformAdmin: u.IsPlatformAdmin, UpdatedAt: u.UpdatedAt,
			},
			OccurredAt: u.UpdatedAt,
		})
	})
	if err != nil {
		return nil, mapUserDomainError(err)
	}
	resp := ToUserResponse(u)
	return &resp, nil
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
