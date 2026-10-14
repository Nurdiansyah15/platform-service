package persistence

import (
	"context"
	"database/sql"
	"errors"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/signingkey/constant"
	"platform-service/internal/domain/signingkey/entity"
	"platform-service/internal/domain/signingkey/valueobject"
)

// PostgresSigningKeyRepository implements repository.SigningKeyRepository.
type PostgresSigningKeyRepository struct {
	db *sql.DB
}

func NewPostgresSigningKeyRepository(db *sql.DB) *PostgresSigningKeyRepository {
	return &PostgresSigningKeyRepository{db: db}
}

const (
	signingKeyColumns           = `id, kid, algorithm, public_key, private_key_ref, status, created_at, retired_at`
	signingKeySingleActiveIndex = "uq_signing_keys_single_active"
)

func (r *PostgresSigningKeyRepository) Save(ctx context.Context, k *entity.SigningKey) error {
	const q = `
		INSERT INTO signing_keys (id, kid, algorithm, public_key, private_key_ref, status, created_at, retired_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			retired_at = EXCLUDED.retired_at`

	_, err := execFromContext(ctx, r.db).ExecContext(ctx, q,
		k.ID, k.KID, k.Algorithm, k.PublicKeyPEM, k.PrivateKeyRef, k.Status.String(), k.CreatedAt, k.RetiredAt)
	if err != nil {
		if isUniqueViolation(err) && constraintName(err) == signingKeySingleActiveIndex {
			return domainerr.New(constant.CodeSigningKeyActiveExists)
		}
		return domainerr.Wrap(constant.CodeSigningKeyPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresSigningKeyRepository) FindActive(ctx context.Context) (*entity.SigningKey, error) {
	return r.findOne(ctx, `SELECT `+signingKeyColumns+` FROM signing_keys WHERE status = 'active'`)
}

func (r *PostgresSigningKeyRepository) FindByKID(ctx context.Context, kid string) (*entity.SigningKey, error) {
	return r.findOne(ctx, `SELECT `+signingKeyColumns+` FROM signing_keys WHERE kid = $1`, kid)
}

func (r *PostgresSigningKeyRepository) ListPublished(ctx context.Context) ([]*entity.SigningKey, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx,
		`SELECT `+signingKeyColumns+` FROM signing_keys WHERE status IN ('active', 'retiring') ORDER BY created_at DESC, kid`)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeSigningKeyQueryFailed, err)
	}
	defer rows.Close()

	var out []*entity.SigningKey
	for rows.Next() {
		k, err := scanSigningKey(rows)
		if err != nil {
			return nil, domainerr.Wrap(constant.CodeSigningKeyQueryFailed, err)
		}
		out = append(out, k)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeSigningKeyQueryFailed, err)
	}
	return out, nil
}

func (r *PostgresSigningKeyRepository) findOne(ctx context.Context, q string, args ...any) (*entity.SigningKey, error) {
	k, err := scanSigningKey(execFromContext(ctx, r.db).QueryRowContext(ctx, q, args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeSigningKeyNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeSigningKeyQueryFailed, err)
	}
	return k, nil
}

func scanSigningKey(row interface{ Scan(dest ...any) error }) (*entity.SigningKey, error) {
	var (
		k      entity.SigningKey
		status string
	)
	if err := row.Scan(&k.ID, &k.KID, &k.Algorithm, &k.PublicKeyPEM, &k.PrivateKeyRef, &status, &k.CreatedAt, &k.RetiredAt); err != nil {
		return nil, err
	}
	var err error
	if k.Status, err = valueobject.NewKeyStatus(status); err != nil {
		return nil, err
	}
	return &k, nil
}
