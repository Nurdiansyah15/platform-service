package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/domain/benefit/constant"
	domainerr "platform-service/internal/domain/errors"
)

// PostgresBenefitQuery implements port.BenefitQueryReadModel.
type PostgresBenefitQuery struct {
	db *sql.DB
}

func NewPostgresBenefitQuery(db *sql.DB) *PostgresBenefitQuery {
	return &PostgresBenefitQuery{db: db}
}

var benefitSortable = map[string]string{"code": "code", "name": "name", "created_at": "created_at"}

func (r *PostgresBenefitQuery) List(ctx context.Context, q port.BenefitListReadQuery) ([]dto.BenefitResponse, dto.Meta, error) {
	limit, offset, page, sortColumn, sortType := resolvePaginationParams(q.PaginationParams, 50, 200, benefitSortable, "code", "asc")

	var w whereBuilder
	w.add(`solution_id = $%d`, q.SolutionID)
	if q.ValueType != nil {
		w.add(`value_type = $%d`, *q.ValueType)
	}
	if q.Metering != nil {
		w.add(`metering = $%d`, *q.Metering)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		w.add(`(code ILIKE $%d OR name ILIKE $%d)`, likePattern(*q.Search), likePattern(*q.Search))
	}

	exec := execFromContext(ctx, r.db)
	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) FROM benefits `+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeBenefitQueryFailed, err)
	}

	rows, err := exec.QueryContext(ctx, fmt.Sprintf(`SELECT %s FROM benefits %s ORDER BY %s %s, id LIMIT %d OFFSET %d`,
		benefitColumns, w.sql(), sortColumn, sortType, limit, offset), w.args...)
	if err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeBenefitQueryFailed, err)
	}
	defer rows.Close()

	items := make([]dto.BenefitResponse, 0, limit)
	for rows.Next() {
		var it dto.BenefitResponse
		if err := rows.Scan(&it.ID, &it.SolutionID, &it.Code, &it.Name, &it.Description, &it.ValueType,
			&it.AggregationStrategy, &it.Unit, &it.Metering, &it.UsageReset, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodeBenefitQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeBenefitQueryFailed, err)
	}
	return items, dto.NewMeta(page, limit, total), nil
}
