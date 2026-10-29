package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/solutionclient/constant"
)

// PostgresSolutionClientQuery implements port.SolutionClientQueryReadModel.
type PostgresSolutionClientQuery struct {
	db *sql.DB
	// typeMap dipakai untuk scan kolom TEXT[] (database/sql tidak mendukung array).
	typeMap *pgtype.Map
}

func NewPostgresSolutionClientQuery(db *sql.DB) *PostgresSolutionClientQuery {
	return &PostgresSolutionClientQuery{db: db, typeMap: pgtype.NewMap()}
}

var solutionClientSortable = map[string]string{"name": "name", "created_at": "created_at", "last_used_at": "last_used_at"}

func (r *PostgresSolutionClientQuery) List(ctx context.Context, q port.SolutionClientListReadQuery) ([]dto.SolutionClientResponse, dto.Meta, error) {
	limit, offset, page, sortColumn, sortType := resolvePaginationParams(q.PaginationParams, 20, 100, solutionClientSortable, "created_at", "desc")

	var w whereBuilder
	w.add(`solution_id = $%d`, q.SolutionID)
	if q.Status != nil {
		w.add(`status = $%d`, *q.Status)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		w.add(`(name ILIKE $%d OR client_id ILIKE $%d)`, likePattern(*q.Search), likePattern(*q.Search))
	}

	exec := execFromContext(ctx, r.db)
	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) FROM solution_clients `+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeClientQueryFailed, err)
	}
	rows, err := exec.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, solution_id, name, client_id, scopes, status, last_used_at, created_at, updated_at, revoked_at
		FROM solution_clients %s ORDER BY %s %s, id LIMIT %d OFFSET %d`,
		w.sql(), sortColumn, sortType, limit, offset), w.args...)
	if err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeClientQueryFailed, err)
	}
	defer rows.Close()

	items := make([]dto.SolutionClientResponse, 0, limit)
	for rows.Next() {
		var it dto.SolutionClientResponse
		if err := rows.Scan(&it.ID, &it.SolutionID, &it.Name, &it.ClientID, r.typeMap.SQLScanner(&it.Scopes), &it.Status,
			&it.LastUsedAt, &it.CreatedAt, &it.UpdatedAt, &it.RevokedAt); err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodeClientQueryFailed, err)
		}
		if it.Scopes == nil {
			it.Scopes = []string{}
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeClientQueryFailed, err)
	}
	return items, dto.NewMeta(page, limit, total), nil
}
