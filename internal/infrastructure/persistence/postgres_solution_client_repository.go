package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/solutionclient/constant"
	"platform-service/internal/domain/solutionclient/entity"
	"platform-service/internal/domain/solutionclient/valueobject"
)

// PostgresSolutionClientRepository implements repository.SolutionClientRepository.
type PostgresSolutionClientRepository struct {
	db *sql.DB
	// typeMap dipakai untuk scan kolom TEXT[] (database/sql tidak mendukung array).
	typeMap *pgtype.Map
}

func NewPostgresSolutionClientRepository(db *sql.DB) *PostgresSolutionClientRepository {
	return &PostgresSolutionClientRepository{db: db, typeMap: pgtype.NewMap()}
}

const solutionClientColumns = `id, solution_id, name, client_id, client_secret_hash, scopes, status,
	last_used_at, created_at, updated_at, revoked_at`

func (r *PostgresSolutionClientRepository) Save(ctx context.Context, c *entity.SolutionClient) error {
	const q = `
		INSERT INTO solution_clients (id, solution_id, name, client_id, client_secret_hash, scopes, status,
			last_used_at, created_at, updated_at, revoked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			client_secret_hash = EXCLUDED.client_secret_hash,
			scopes = EXCLUDED.scopes,
			status = EXCLUDED.status,
			revoked_at = EXCLUDED.revoked_at
		RETURNING created_at, updated_at`

	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		c.ID, c.SolutionID, c.Name, c.ClientID, c.ClientSecretHash, c.Scopes.Values(), c.Status.String(),
		c.LastUsedAt, c.CreatedAt, c.UpdatedAt, c.RevokedAt,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		// client_id acak 96 bit sehingga bentrok unique praktis mustahil; semua error = gagal simpan.
		return domainerr.Wrap(constant.CodeClientPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresSolutionClientRepository) FindByID(ctx context.Context, id string) (*entity.SolutionClient, error) {
	return r.findOne(ctx, `SELECT `+solutionClientColumns+` FROM solution_clients WHERE id = $1`, id)
}

func (r *PostgresSolutionClientRepository) FindByClientID(ctx context.Context, clientID string) (*entity.SolutionClient, error) {
	return r.findOne(ctx, `SELECT `+solutionClientColumns+` FROM solution_clients WHERE client_id = $1`, clientID)
}

func (r *PostgresSolutionClientRepository) TouchLastUsed(ctx context.Context, id string, at time.Time) error {
	if _, err := execFromContext(ctx, r.db).ExecContext(ctx,
		`UPDATE solution_clients SET last_used_at = $2 WHERE id = $1`, id, at.UTC()); err != nil {
		return domainerr.Wrap(constant.CodeClientPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresSolutionClientRepository) findOne(ctx context.Context, q string, args ...any) (*entity.SolutionClient, error) {
	var (
		c      entity.SolutionClient
		scopes []string
		status string
	)
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q, args...).Scan(
		&c.ID, &c.SolutionID, &c.Name, &c.ClientID, &c.ClientSecretHash, r.typeMap.SQLScanner(&scopes), &status,
		&c.LastUsedAt, &c.CreatedAt, &c.UpdatedAt, &c.RevokedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeClientNotFound)
		}
		return nil, domainerr.Wrap(constant.CodeClientQueryFailed, err)
	}
	if c.Scopes, err = valueobject.NewScopes(scopes); err != nil {
		return nil, domainerr.Wrap(constant.CodeClientQueryFailed, err)
	}
	if c.Status, err = valueobject.NewClientStatus(status); err != nil {
		return nil, domainerr.Wrap(constant.CodeClientQueryFailed, err)
	}
	return &c, nil
}
