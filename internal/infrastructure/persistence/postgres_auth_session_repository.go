package persistence

import (
	"context"
	"database/sql"
	"errors"

	"platform-service/internal/domain/authsession/constant"
	"platform-service/internal/domain/authsession/entity"
	domainerr "platform-service/internal/domain/errors"
)

// PostgresAuthSessionRepository implements repository.AuthSessionRepository.
type PostgresAuthSessionRepository struct {
	db *sql.DB
}

func NewPostgresAuthSessionRepository(db *sql.DB) *PostgresAuthSessionRepository {
	return &PostgresAuthSessionRepository{db: db}
}

const authSessionColumns = `id, user_id, solution_id, company_id, refresh_token_hash, user_agent,
	host(ip_address), expires_at, last_used_at, revoked_at, created_at`

func (r *PostgresAuthSessionRepository) Create(ctx context.Context, s *entity.AuthSession) error {
	// NULLIF: c.ClientIP() bisa "" (mis. httptest) dan ''::inet akan error.
	const q = `
		INSERT INTO auth_sessions (id, user_id, solution_id, company_id, refresh_token_hash, user_agent,
			ip_address, expires_at, last_used_at, revoked_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::inet, $8, $9, $10, $11)`

	var ip string
	if s.IPAddress != nil {
		ip = *s.IPAddress
	}
	_, err := execFromContext(ctx, r.db).ExecContext(ctx, q,
		s.ID, s.UserID, s.SolutionID, s.CompanyID, s.RefreshTokenHash, s.UserAgent,
		ip, s.ExpiresAt, s.LastUsedAt, s.RevokedAt, s.CreatedAt)
	if err != nil {
		return domainerr.Wrap(constant.CodeSessionPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresAuthSessionRepository) FindByID(ctx context.Context, id string) (*entity.AuthSession, error) {
	return r.findOne(ctx, `SELECT `+authSessionColumns+` FROM auth_sessions WHERE id = $1`, id)
}

func (r *PostgresAuthSessionRepository) FindByRefreshTokenHash(ctx context.Context, hash string) (*entity.AuthSession, error) {
	return r.findOne(ctx, `SELECT `+authSessionColumns+` FROM auth_sessions WHERE refresh_token_hash = $1`, hash)
}

func (r *PostgresAuthSessionRepository) SaveRotation(ctx context.Context, s *entity.AuthSession, previousHash string) error {
	const q = `
		UPDATE auth_sessions
		SET refresh_token_hash = $2, expires_at = $3, last_used_at = $4, company_id = $6
		WHERE id = $1 AND refresh_token_hash = $5 AND revoked_at IS NULL`

	res, err := execFromContext(ctx, r.db).ExecContext(ctx, q, s.ID, s.RefreshTokenHash, s.ExpiresAt, s.LastUsedAt, previousHash, s.CompanyID)
	if err != nil {
		return domainerr.Wrap(constant.CodeSessionPersistenceFailed, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return domainerr.Wrap(constant.CodeSessionPersistenceFailed, err)
	}
	if n == 0 {
		return domainerr.New(constant.CodeSessionRotationConflict)
	}
	return nil
}

func (r *PostgresAuthSessionRepository) SaveRevocation(ctx context.Context, s *entity.AuthSession) error {
	const q = `UPDATE auth_sessions SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`
	if _, err := execFromContext(ctx, r.db).ExecContext(ctx, q, s.ID, s.RevokedAt); err != nil {
		return domainerr.Wrap(constant.CodeSessionPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresAuthSessionRepository) findOne(ctx context.Context, q string, args ...any) (*entity.AuthSession, error) {
	var s entity.AuthSession
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q, args...).Scan(
		&s.ID, &s.UserID, &s.SolutionID, &s.CompanyID, &s.RefreshTokenHash, &s.UserAgent,
		&s.IPAddress, &s.ExpiresAt, &s.LastUsedAt, &s.RevokedAt, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeSessionNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeSessionQueryFailed, err)
	}
	return &s, nil
}
