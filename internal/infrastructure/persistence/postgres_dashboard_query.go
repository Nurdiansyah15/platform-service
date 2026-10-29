package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"platform-service/internal/app/dto"
)

// PostgresDashboardQuery implements port.DashboardQueryReadModel.
type PostgresDashboardQuery struct {
	db *sql.DB
}

func NewPostgresDashboardQuery(db *sql.DB) *PostgresDashboardQuery {
	return &PostgresDashboardQuery{db: db}
}

func (r *PostgresDashboardQuery) AdminSummary(ctx context.Context) (*dto.AdminSummaryResponse, error) {
	var s dto.AdminSummaryResponse
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM users WHERE status = 'active'),
			(SELECT COUNT(*) FROM users WHERE status = 'inactive'),
			(SELECT COUNT(*) FROM users WHERE status = 'suspended'),
			(SELECT COUNT(*) FROM users WHERE is_platform_admin),
			(SELECT COUNT(*) FROM companies WHERE status = 'active'),
			(SELECT COUNT(*) FROM companies WHERE status = 'suspended'),
			(SELECT COUNT(*) FROM companies WHERE status = 'closed'),
			(SELECT COUNT(*) FROM solutions WHERE status = 'active'),
			(SELECT COUNT(*) FROM solutions WHERE status = 'inactive'),
			(SELECT COUNT(*) FROM solution_clients WHERE status = 'active')`,
	).Scan(&s.Users.Active, &s.Users.Inactive, &s.Users.Suspended, &s.Users.Admins,
		&s.Companies.Active, &s.Companies.Suspended, &s.Companies.Closed,
		&s.Solutions.Active, &s.Solutions.Inactive, &s.ActiveClients)
	if err != nil {
		// Query lintas domain tanpa error code domain sendiri; dipetakan ke 500 oleh usecase.
		return nil, fmt.Errorf("dashboard: admin summary: %w", err)
	}
	return &s, nil
}
