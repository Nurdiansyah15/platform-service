package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/organization/constant"
)

// PostgresCompanyQuery implements port.CompanyQueryReadModel.
type PostgresCompanyQuery struct {
	db *sql.DB
}

func NewPostgresCompanyQuery(db *sql.DB) *PostgresCompanyQuery {
	return &PostgresCompanyQuery{db: db}
}

var (
	companySortable = map[string]string{"name": "c.name", "slug": "c.slug", "created_at": "c.created_at"}
	memberSortable  = map[string]string{"email": "u.email", "name": "u.name", "joined_at": "m.joined_at"}
)

// whereBuilder mengumpulkan kondisi WHERE dengan placeholder berurutan.
type whereBuilder struct {
	conds []string
	args  []any
}

func (b *whereBuilder) add(cond string, vals ...any) {
	ph := make([]any, len(vals))
	for i, v := range vals {
		b.args = append(b.args, v)
		ph[i] = len(b.args)
	}
	b.conds = append(b.conds, fmt.Sprintf(cond, ph...))
}

func (b *whereBuilder) sql() string {
	if len(b.conds) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(b.conds, " AND ")
}

func likePattern(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(strings.TrimSpace(s)) + "%"
}

func (r *PostgresCompanyQuery) ListCompanies(ctx context.Context, q port.CompanyListReadQuery) ([]port.CompanyReadItem, dto.Meta, error) {
	limit, offset, page, sortColumn, sortType := resolvePaginationParams(q.PaginationParams, 20, 100, companySortable, "c.name", "asc")

	var w whereBuilder
	if q.MemberUserID != nil {
		w.add(`EXISTS (SELECT 1 FROM company_members m WHERE m.company_id = c.id AND m.user_id = $%d AND m.status = 'active')`, *q.MemberUserID)
	}
	if q.Status != nil {
		w.add(`c.status = $%d`, *q.Status)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		w.add(`(c.name ILIKE $%d OR c.slug ILIKE $%d)`, likePattern(*q.Search), likePattern(*q.Search))
	}

	exec := execFromContext(ctx, r.db)
	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) FROM companies c `+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
	}

	rows, err := exec.QueryContext(ctx, fmt.Sprintf(
		`SELECT c.id, c.name, c.slug, c.status, c.created_at, c.updated_at,
			(SELECT COUNT(*) FROM company_members cm WHERE cm.company_id = c.id AND cm.status = 'active')
		 FROM companies c %s
		 ORDER BY %s %s, c.id LIMIT %d OFFSET %d`, w.sql(), sortColumn, sortType, limit, offset), w.args...)
	if err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
	}
	defer rows.Close()

	items := make([]port.CompanyReadItem, 0, limit)
	for rows.Next() {
		var it port.CompanyReadItem
		if err := rows.Scan(&it.ID, &it.Name, &it.Slug, &it.Status, &it.CreatedAt, &it.UpdatedAt, &it.MemberCount); err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
	}
	return items, dto.NewMeta(page, limit, total), nil
}

func (r *PostgresCompanyQuery) ListMembers(ctx context.Context, q port.CompanyMemberListReadQuery) ([]port.CompanyMemberReadItem, dto.Meta, error) {
	limit, offset, page, sortColumn, sortType := resolvePaginationParams(q.PaginationParams, 20, 100, memberSortable, "m.joined_at", "asc")

	var w whereBuilder
	w.add(`m.company_id = $%d`, q.CompanyID)
	if q.Status != nil {
		w.add(`m.status = $%d`, *q.Status)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		w.add(`(u.email ILIKE $%d OR u.name ILIKE $%d)`, likePattern(*q.Search), likePattern(*q.Search))
	}
	from := `FROM company_members m JOIN users u ON u.id = m.user_id ` + w.sql()

	exec := execFromContext(ctx, r.db)
	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) `+from, w.args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
	}

	rows, err := exec.QueryContext(ctx, fmt.Sprintf(
		`SELECT m.user_id, u.email, u.name, m.status, m.joined_at,
			(SELECT COUNT(*) FROM user_role_assignments ura WHERE ura.company_id = m.company_id AND ura.user_id = m.user_id)
		 %s ORDER BY %s %s, m.user_id LIMIT %d OFFSET %d`,
		from, sortColumn, sortType, limit, offset), w.args...)
	if err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
	}
	defer rows.Close()

	items := make([]port.CompanyMemberReadItem, 0, limit)
	for rows.Next() {
		var it port.CompanyMemberReadItem
		if err := rows.Scan(&it.UserID, &it.Email, &it.Name, &it.Status, &it.JoinedAt, &it.RoleCount); err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
	}
	return items, dto.NewMeta(page, limit, total), nil
}

func (r *PostgresCompanyQuery) SearchMemberCandidates(ctx context.Context, companyID, search string, limit int) ([]dto.MemberCandidateResponse, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx, `
		SELECT u.id, u.name, u.email
		FROM users u
		WHERE u.status = 'active'
		  AND (u.email ILIKE $2 OR u.name ILIKE $2)
		  AND NOT EXISTS (SELECT 1 FROM company_members m
		                  WHERE m.company_id = $1 AND m.user_id = u.id AND m.status IN ('active', 'invited'))
		ORDER BY u.name, u.email
		LIMIT $3`, companyID, likePattern(search), limit)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
	}
	defer rows.Close()
	items := []dto.MemberCandidateResponse{}
	for rows.Next() {
		var it dto.MemberCandidateResponse
		if err := rows.Scan(&it.ID, &it.Name, &it.Email); err != nil {
			return nil, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
	}
	return items, nil
}

func (r *PostgresCompanyQuery) Summary(ctx context.Context, companyID string) (*dto.CompanySummaryResponse, error) {
	var s dto.CompanySummaryResponse
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM company_members WHERE company_id = $1 AND status = 'active'),
			(SELECT COUNT(*) FROM company_members WHERE company_id = $1 AND status = 'invited'),
			(SELECT COUNT(*) FROM company_members WHERE company_id = $1 AND status = 'inactive'),
			(SELECT COUNT(*) FROM branches WHERE company_id = $1 AND deleted_at IS NULL),
			(SELECT COUNT(*) FROM roles WHERE company_id = $1 AND deleted_at IS NULL),
			(SELECT COUNT(*) FROM user_role_assignments WHERE company_id = $1)`, companyID,
	).Scan(&s.Members.Active, &s.Members.Invited, &s.Members.Inactive, &s.Branches, &s.CustomRoles, &s.RoleAssignments)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeCompanyQueryFailed, err)
	}
	return &s, nil
}
