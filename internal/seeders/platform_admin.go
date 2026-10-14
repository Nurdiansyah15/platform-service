package seeders

import (
	"context"
	"database/sql"
	"errors"

	"platform-service/internal/app/port"
	"platform-service/internal/app/usecase/user"
	"platform-service/internal/infrastructure/crypto"
	"platform-service/internal/infrastructure/persistence"
)

// PlatformAdminSeeder memastikan satu user platform admin ada. User yang sudah
// ada hanya diberi flag admin; password-nya tidak ditimpa.
type PlatformAdminSeeder struct {
	Email     string
	AdminName string
	Password  string
	// Hasher opsional (nil → bcrypt default); test memakai cost minimum.
	Hasher port.PasswordHasher
}

func (PlatformAdminSeeder) Name() string { return "platform-admin" }

func (s PlatformAdminSeeder) Run(ctx context.Context, db *sql.DB) error {
	if s.Email == "" || s.Password == "" {
		return errors.New("PLATFORM_ADMIN_EMAIL dan PLATFORM_ADMIN_PASSWORD wajib diisi")
	}
	name := s.AdminName
	if name == "" {
		name = "Platform Admin"
	}
	hasher := s.Hasher
	if hasher == nil {
		hasher = crypto.NewBcryptHasher(0)
	}

	uc := user.NewUserUseCase(user.UserDeps{Repo: persistence.NewPostgresUserRepository(db), Hasher: hasher})
	_, err := uc.EnsurePlatformAdmin(ctx, s.Email, name, s.Password)
	return err
}
