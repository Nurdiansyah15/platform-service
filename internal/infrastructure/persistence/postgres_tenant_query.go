package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/tenant/constant"
)

// PostgresTenantQuery implements port.TenantQueryReadModel.
type PostgresTenantQuery struct {
	db *sql.DB
}

func NewPostgresTenantQuery(db *sql.DB) *PostgresTenantQuery {
	return &PostgresTenantQuery{db: db}
}

var (
	tenantSortable = map[string]string{"name": "t.name", "slug": "t.slug", "created_at": "t.created_at"}
	memberSortable = map[string]string{"email": "u.email", "name": "u.name", "joined_at": "m.joined_at"}
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

func (r *PostgresTenantQuery) ListTenants(ctx context.Context, q port.TenantListReadQuery) ([]port.TenantReadItem, dto.Meta, error) {
	limit, offset, page, sortColumn, sortType := resolvePaginationParams(q.PaginationParams, 20, 100, tenantSortable, "t.name", "asc")

	var w whereBuilder
	if q.MemberUserID != nil {
		w.add(`EXISTS (SELECT 1 FROM tenant_members m WHERE m.tenant_id = t.id AND m.user_id = $%d AND m.status = 'active')`, *q.MemberUserID)
	}
	if q.Status != nil {
		w.add(`t.status = $%d`, *q.Status)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		w.add(`(t.name ILIKE $%d OR t.slug ILIKE $%d)`, likePattern(*q.Search), likePattern(*q.Search))
	}

	exec := execFromContext(ctx, r.db)
	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) FROM tenants t `+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeTenantQueryFailed, err)
	}

	rows, err := exec.QueryContext(ctx, fmt.Sprintf(
		`SELECT t.id, t.name, t.slug, t.status, t.created_at, t.updated_at FROM tenants t %s
		 ORDER BY %s %s, t.id LIMIT %d OFFSET %d`, w.sql(), sortColumn, sortType, limit, offset), w.args...)
	if err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeTenantQueryFailed, err)
	}
	defer rows.Close()

	items := make([]port.TenantReadItem, 0, limit)
	for rows.Next() {
		var it port.TenantReadItem
		if err := rows.Scan(&it.ID, &it.Name, &it.Slug, &it.Status, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodeTenantQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeTenantQueryFailed, err)
	}
	return items, dto.NewMeta(page, limit, total), nil
}

func (r *PostgresTenantQuery) ListMembers(ctx context.Context, q port.TenantMemberListReadQuery) ([]port.TenantMemberReadItem, dto.Meta, error) {
	limit, offset, page, sortColumn, sortType := resolvePaginationParams(q.PaginationParams, 20, 100, memberSortable, "m.joined_at", "asc")

	var w whereBuilder
	w.add(`m.tenant_id = $%d`, q.TenantID)
	if q.Status != nil {
		w.add(`m.status = $%d`, *q.Status)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		w.add(`(u.email ILIKE $%d OR u.name ILIKE $%d)`, likePattern(*q.Search), likePattern(*q.Search))
	}
	from := `FROM tenant_members m JOIN users u ON u.id = m.user_id ` + w.sql()

	exec := execFromContext(ctx, r.db)
	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) `+from, w.args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeTenantQueryFailed, err)
	}

	rows, err := exec.QueryContext(ctx, fmt.Sprintf(
		`SELECT m.user_id, u.email, u.name, m.status, m.joined_at %s ORDER BY %s %s, m.user_id LIMIT %d OFFSET %d`,
		from, sortColumn, sortType, limit, offset), w.args...)
	if err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeTenantQueryFailed, err)
	}
	defer rows.Close()

	items := make([]port.TenantMemberReadItem, 0, limit)
	for rows.Next() {
		var it port.TenantMemberReadItem
		if err := rows.Scan(&it.UserID, &it.Email, &it.Name, &it.Status, &it.JoinedAt); err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodeTenantQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeTenantQueryFailed, err)
	}
	return items, dto.NewMeta(page, limit, total), nil
}
