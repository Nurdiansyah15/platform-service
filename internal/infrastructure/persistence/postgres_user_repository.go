package persistence

import (
	"context"
	"database/sql"
	"errors"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/user/constant"
	"platform-service/internal/domain/user/entity"
	"platform-service/internal/domain/user/valueobject"
)

// PostgresUserRepository implements repository.UserRepository.
type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

const userColumns = `id, email, name, password_hash, status, is_platform_admin, created_at, updated_at`

func (r *PostgresUserRepository) Save(ctx context.Context, u *entity.User) error {
	const q = `
		INSERT INTO users (id, email, name, password_hash, status, is_platform_admin, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			password_hash = EXCLUDED.password_hash,
			status = EXCLUDED.status,
			is_platform_admin = EXCLUDED.is_platform_admin
		RETURNING created_at, updated_at`

	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		u.ID, u.Email.Value(), u.Name, u.PasswordHash, u.Status.String(), u.IsPlatformAdmin, u.CreatedAt, u.UpdatedAt,
	).Scan(&u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodeUserEmailDuplicate)
		}
		return domainerr.Wrap(constant.CodeUserPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	return r.findOne(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	normalized, err := valueobject.NewEmail(email)
	if err != nil {
		return nil, domainerr.New(constant.CodeUserNotFound)
	}
	return r.findOne(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, normalized.Value())
}

func (r *PostgresUserRepository) findOne(ctx context.Context, q string, args ...any) (*entity.User, error) {
	var (
		u             entity.User
		email, status string
	)
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q, args...).
		Scan(&u.ID, &email, &u.Name, &u.PasswordHash, &status, &u.IsPlatformAdmin, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeUserNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeUserQueryFailed, err)
	}
	if u.Email, err = valueobject.NewEmail(email); err != nil {
		return nil, domainerr.Wrap(constant.CodeUserQueryFailed, err)
	}
	if u.Status, err = valueobject.NewUserStatus(status); err != nil {
		return nil, domainerr.Wrap(constant.CodeUserQueryFailed, err)
	}
	return &u, nil
}
