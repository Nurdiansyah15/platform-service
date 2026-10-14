package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	benefitconstant "platform-service/internal/domain/benefit/constant"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/plan/constant"
)

// benefitSolutionMismatch dipakai repository plan/addon saat composite FK (benefit_id, solution_id) ditolak DB.
const benefitSolutionMismatch = benefitconstant.CodeBenefitSolutionMismatch

// PostgresPlanQuery implements port.PlanQueryReadModel.
type PostgresPlanQuery struct {
	db *sql.DB
}

func NewPostgresPlanQuery(db *sql.DB) *PostgresPlanQuery {
	return &PostgresPlanQuery{db: db}
}

// Kolom sort di-qualify "plans." karena output price::text bernama "price" dan
// ORDER BY nama polos akan merujuk kolom output (pengurutan teks, bukan numerik).
var planSortable = map[string]string{"code": "plans.code", "name": "plans.name", "price": "plans.price", "created_at": "plans.created_at"}

func (r *PostgresPlanQuery) ListPlans(ctx context.Context, q port.PlanListReadQuery) ([]port.PlanReadItem, dto.Meta, error) {
	limit, offset, page, sortColumn, sortType := resolvePaginationParams(q.PaginationParams, 20, 100, planSortable, "plans.price", "asc")

	var w whereBuilder
	w.add(`solution_id = $%d`, q.SolutionID)
	w.add(`deleted_at IS NULL`)
	if q.IsActive != nil {
		w.add(`is_active = $%d`, *q.IsActive)
	}
	if q.BillingCycle != nil {
		w.add(`billing_cycle = $%d`, *q.BillingCycle)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		w.add(`(code ILIKE $%d OR name ILIKE $%d)`, likePattern(*q.Search), likePattern(*q.Search))
	}

	exec := execFromContext(ctx, r.db)
	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) FROM plans `+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodePlanQueryFailed, err)
	}

	rows, err := exec.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, solution_id, code, name, description, price::text, currency, billing_cycle, is_active, created_at, updated_at
		FROM plans %s ORDER BY %s %s, plans.id LIMIT %d OFFSET %d`, w.sql(), sortColumn, sortType, limit, offset), w.args...)
	if err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodePlanQueryFailed, err)
	}
	defer rows.Close()

	items := make([]port.PlanReadItem, 0, limit)
	for rows.Next() {
		var it port.PlanReadItem
		if err := rows.Scan(&it.ID, &it.SolutionID, &it.Code, &it.Name, &it.Description, &it.Price, &it.Currency,
			&it.BillingCycle, &it.IsActive, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodePlanQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodePlanQueryFailed, err)
	}
	return items, dto.NewMeta(page, limit, total), nil
}

func (r *PostgresPlanQuery) ListPlanBenefits(ctx context.Context, planID string) ([]port.AssignedBenefitReadItem, error) {
	return listAssignedBenefits(ctx, execFromContext(ctx, r.db), `
		SELECT b.id, b.code, b.name, b.value_type, b.aggregation_strategy, b.unit, b.metering, pb.value
		FROM plan_benefits pb JOIN benefits b ON b.id = pb.benefit_id
		WHERE pb.plan_id = $1 ORDER BY b.code`, planID, constant.CodePlanQueryFailed)
}

// listAssignedBenefits dipakai bersama plan & add-on.
func listAssignedBenefits(ctx context.Context, exec executor, q, parentID string, failCode domainerr.Code) ([]port.AssignedBenefitReadItem, error) {
	rows, err := exec.QueryContext(ctx, q, parentID)
	if err != nil {
		return nil, domainerr.Wrap(failCode, err)
	}
	defer rows.Close()

	items := []port.AssignedBenefitReadItem{}
	for rows.Next() {
		var (
			it    port.AssignedBenefitReadItem
			value []byte
		)
		if err := rows.Scan(&it.BenefitID, &it.Code, &it.Name, &it.ValueType, &it.AggregationStrategy, &it.Unit, &it.Metering, &value); err != nil {
			return nil, domainerr.Wrap(failCode, err)
		}
		it.Value = value
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(failCode, err)
	}
	return items, nil
}
