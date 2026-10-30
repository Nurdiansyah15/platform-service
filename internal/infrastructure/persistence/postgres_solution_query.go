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
	"platform-service/internal/domain/solution/constant"
)

// PostgresSolutionQuery implements port.SolutionQueryReadModel.
type PostgresSolutionQuery struct {
	db      *sql.DB
	typeMap *pgtype.Map
}

func NewPostgresSolutionQuery(db *sql.DB) *PostgresSolutionQuery {
	return &PostgresSolutionQuery{db: db, typeMap: pgtype.NewMap()}
}

var solutionSortable = map[string]string{
	"code":       "code",
	"name":       "name",
	"created_at": "created_at",
}

func (r *PostgresSolutionQuery) List(ctx context.Context, q port.SolutionListReadQuery) ([]port.SolutionReadItem, dto.Meta, error) {
	limit, offset, page, sortColumn, sortType := resolvePaginationParams(q.PaginationParams, 20, 100, solutionSortable, "name", "asc")

	var (
		where []string
		args  []any
	)
	if q.Status != nil {
		args = append(args, *q.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		args = append(args, likePattern(*q.Search))
		where = append(where, fmt.Sprintf("(code ILIKE $%d OR name ILIKE $%d)", len(args), len(args)))
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	exec := execFromContext(ctx, r.db)

	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) FROM solutions `+whereSQL, args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeSolutionQueryFailed, err)
	}

	listSQL := fmt.Sprintf(`SELECT %s FROM solutions %s ORDER BY %s %s, id LIMIT %d OFFSET %d`,
		solutionColumns, whereSQL, sortColumn, sortType, limit, offset)
	rows, err := exec.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeSolutionQueryFailed, err)
	}
	defer rows.Close()

	items := make([]port.SolutionReadItem, 0, limit)
	for rows.Next() {
		var it port.SolutionReadItem
		if err := rows.Scan(&it.ID, &it.Code, &it.Name, &it.Description, &it.Status, &it.PermissionCatalogVersion,
			&it.LaunchURL, &it.IconURL, r.typeMap.SQLScanner(&it.RedirectURIs), &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodeSolutionQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeSolutionQueryFailed, err)
	}

	return items, dto.NewMeta(page, limit, total), nil
}
