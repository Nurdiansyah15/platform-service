package persistence

import (
	"context"
	"database/sql"
	"errors"

	"platform-service/internal/domain/authsession/constant"
	"platform-service/internal/domain/authsession/entity"
	domainerr "platform-service/internal/domain/errors"
)

// PostgresAuthorizationCodeRepository implements repository.AuthorizationCodeRepository.
type PostgresAuthorizationCodeRepository struct {
	db *sql.DB
}

func NewPostgresAuthorizationCodeRepository(db *sql.DB) *PostgresAuthorizationCodeRepository {
	return &PostgresAuthorizationCodeRepository{db: db}
}

const authorizationCodeColumns = `id, code_hash, user_id, solution_id, company_id, parent_session_id, redirect_uri,
	code_challenge, code_challenge_method, expires_at, used_at, created_at`

func (r *PostgresAuthorizationCodeRepository) Create(ctx context.Context, c *entity.AuthorizationCode) error {
	const q = `
		INSERT INTO auth_authorization_codes (` + authorizationCodeColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := execFromContext(ctx, r.db).ExecContext(ctx, q,
		c.ID, c.CodeHash, c.UserID, c.SolutionID, c.CompanyID, c.ParentSessionID, c.RedirectURI,
		c.CodeChallenge, c.CodeChallengeMethod, c.ExpiresAt, c.UsedAt, c.CreatedAt)
	if err != nil {
		return domainerr.Wrap(constant.CodeAuthCodePersistenceFailed, err)
	}
	return nil
}

func (r *PostgresAuthorizationCodeRepository) FindByCodeHash(ctx context.Context, hash string) (*entity.AuthorizationCode, error) {
	var c entity.AuthorizationCode
	err := execFromContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT `+authorizationCodeColumns+` FROM auth_authorization_codes WHERE code_hash = $1`, hash).Scan(
		&c.ID, &c.CodeHash, &c.UserID, &c.SolutionID, &c.CompanyID, &c.ParentSessionID, &c.RedirectURI,
		&c.CodeChallenge, &c.CodeChallengeMethod, &c.ExpiresAt, &c.UsedAt, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeAuthCodeNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeAuthCodeQueryFailed, err)
	}
	return &c, nil
}

func (r *PostgresAuthorizationCodeRepository) MarkUsed(ctx context.Context, c *entity.AuthorizationCode) error {
	const q = `UPDATE auth_authorization_codes SET used_at = $2 WHERE id = $1 AND used_at IS NULL`
	res, err := execFromContext(ctx, r.db).ExecContext(ctx, q, c.ID, c.UsedAt)
	if err != nil {
		return domainerr.Wrap(constant.CodeAuthCodePersistenceFailed, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return domainerr.Wrap(constant.CodeAuthCodePersistenceFailed, err)
	}
	if n == 0 {
		return domainerr.New(constant.CodeAuthCodeAlreadyUsed)
	}
	return nil
}
