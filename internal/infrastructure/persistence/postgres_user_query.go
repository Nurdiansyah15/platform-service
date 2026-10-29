package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/user/constant"
)

// PostgresUserQuery implements port.UserQueryReadModel.
type PostgresUserQuery struct {
	db *sql.DB
}

func NewPostgresUserQuery(db *sql.DB) *PostgresUserQuery {
	return &PostgresUserQuery{db: db}
}

var userSortable = map[string]string{"email": "email", "name": "name", "created_at": "created_at"}

func (r *PostgresUserQuery) List(ctx context.Context, q port.UserListReadQuery) ([]dto.UserResponse, dto.Meta, error) {
	limit, offset, page, sortColumn, sortType := resolvePaginationParams(q.PaginationParams, 20, 100, userSortable, "created_at", "desc")

	var w whereBuilder
	if q.Status != nil {
		w.add(`status = $%d`, *q.Status)
	}
	if q.IsPlatformAdmin != nil {
		w.add(`is_platform_admin = $%d`, *q.IsPlatformAdmin)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		w.add(`(email ILIKE $%d OR name ILIKE $%d)`, likePattern(*q.Search), likePattern(*q.Search))
	}

	exec := execFromContext(ctx, r.db)
	var total int64
	if err := exec.QueryRowContext(ctx, `SELECT COUNT(*) FROM users `+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeUserQueryFailed, err)
	}
	rows, err := exec.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, email, name, status, is_platform_admin, password_hash IS NOT NULL AND password_hash <> '', created_at, updated_at
		FROM users %s ORDER BY %s %s, id LIMIT %d OFFSET %d`,
		w.sql(), sortColumn, sortType, limit, offset), w.args...)
	if err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeUserQueryFailed, err)
	}
	defer rows.Close()

	items := make([]dto.UserResponse, 0, limit)
	for rows.Next() {
		var it dto.UserResponse
		if err := rows.Scan(&it.ID, &it.Email, &it.Name, &it.Status, &it.IsPlatformAdmin, &it.HasPassword, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, dto.Meta{}, domainerr.Wrap(constant.CodeUserQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, dto.Meta{}, domainerr.Wrap(constant.CodeUserQueryFailed, err)
	}
	return items, dto.NewMeta(page, limit, total), nil
}

func (r *PostgresUserQuery) ListMemberships(ctx context.Context, userID string) ([]dto.UserMembershipResponse, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx, `
		SELECT c.id, c.name, c.slug, c.status, m.status, m.joined_at
		FROM company_members m
		JOIN companies c ON c.id = m.company_id
		WHERE m.user_id = $1
		ORDER BY c.name, c.id`, userID)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodeUserQueryFailed, err)
	}
	defer rows.Close()

	items := []dto.UserMembershipResponse{}
	for rows.Next() {
		var it dto.UserMembershipResponse
		if err := rows.Scan(&it.CompanyID, &it.CompanyName, &it.CompanySlug, &it.CompanyStatus, &it.MemberStatus, &it.JoinedAt); err != nil {
			return nil, domainerr.Wrap(constant.CodeUserQueryFailed, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodeUserQueryFailed, err)
	}
	return items, nil
}
