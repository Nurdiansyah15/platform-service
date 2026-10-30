package seeders

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	authzconstant "platform-service/internal/domain/authz/constant"
)

// ConsoleSolutionSeeder memastikan solution "console" ada beserta katalog permission
// platform (administrasi company) dan role sistem company_admin. Solution ini dipakai
// sebagai audience (aud) sesi login console dan sebagai pemilik permission platform.
// Idempotent: nama / status solution yang sudah ada tidak diubah.
//
// Saat katalog dipasang pertama kali (permission_catalog_version masih kosong), setiap
// member active dari company yang ada diberi company_admin agar perilaku lama — semua
// member boleh mengelola company — tidak berubah. Jalankan ulang tidak memberi admin lagi.
type ConsoleSolutionSeeder struct {
	Code         string
	SolutionName string
}

func (ConsoleSolutionSeeder) Name() string { return "console-solution" }

func (s ConsoleSolutionSeeder) Run(ctx context.Context, db *sql.DB) error {
	if s.Code == "" {
		return errors.New("CONSOLE_SOLUTION_CODE wajib diisi")
	}
	name := s.SolutionName
	if name == "" {
		name = s.Code
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO solutions (code, name, status)
		VALUES ($1, $2, 'active')
		ON CONFLICT (code) DO NOTHING`, s.Code, name); err != nil {
		return err
	}
	var (
		solutionID  string
		prevVersion sql.NullString
	)
	if err := tx.QueryRowContext(ctx,
		`SELECT id, permission_catalog_version FROM solutions WHERE code = $1 FOR UPDATE`, s.Code,
	).Scan(&solutionID, &prevVersion); err != nil {
		return err
	}

	codes := make([]string, 0, len(authzconstant.PlatformPermissions))
	for _, p := range authzconstant.PlatformPermissions {
		codes = append(codes, p.Code)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO permissions (solution_id, code, name, description, status)
			VALUES ($1, $2, $3, $4, 'active')
			ON CONFLICT (solution_id, code) DO UPDATE SET
				name = EXCLUDED.name, description = EXCLUDED.description, status = 'active'`,
			solutionID, p.Code, p.Name, p.Description); err != nil {
			return fmt.Errorf("permission %s: %w", p.Code, err)
		}
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE solutions SET permission_catalog_version = $2 WHERE id = $1`,
		solutionID, authzconstant.PlatformCatalogVersion); err != nil {
		return err
	}

	roleID, err := ensureCompanyAdminRole(ctx, tx, solutionID, codes)
	if err != nil {
		return err
	}

	if !prevVersion.Valid {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_role_assignments (company_id, user_id, role_id, scope_type, scope_id)
			SELECT m.company_id, m.user_id, $1, 'company', m.company_id
			FROM company_members m
			WHERE m.status = 'active'
			ON CONFLICT (user_id, role_id, scope_type, scope_id) DO NOTHING`, roleID); err != nil {
			return fmt.Errorf("backfill company_admin: %w", err)
		}
	}
	return tx.Commit()
}

// ensureCompanyAdminRole membuat (bila belum ada) role global sistem company_admin dan
// menyamakan permission-nya dengan katalog platform.
func ensureCompanyAdminRole(ctx context.Context, tx *sql.Tx, solutionID string, codes []string) (string, error) {
	var roleID string
	err := tx.QueryRowContext(ctx, `
		SELECT id FROM roles
		WHERE solution_id = $1 AND company_id IS NULL AND code = $2 AND deleted_at IS NULL`,
		solutionID, authzconstant.CompanyAdminRoleCode).Scan(&roleID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO roles (solution_id, code, name, description, is_system)
			VALUES ($1, $2, 'Admin company', 'Mengelola info company, anggota, cabang, role, dan penugasan role.', TRUE)
			RETURNING id`, solutionID, authzconstant.CompanyAdminRoleCode).Scan(&roleID); err != nil {
			return "", fmt.Errorf("role company_admin: %w", err)
		}
	case err != nil:
		return "", err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO role_permissions (role_id, permission_id, solution_id)
		SELECT $1, p.id, p.solution_id FROM permissions p
		WHERE p.solution_id = $2 AND p.code = ANY($3)`, roleID, solutionID, codes); err != nil {
		return "", fmt.Errorf("permission company_admin: %w", err)
	}
	return roleID, nil
}
