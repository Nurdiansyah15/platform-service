package seeders

import (
	"context"
	"database/sql"
	"errors"
)

// ConsoleSolutionSeeder memastikan satu solution "console" ada. Solution ini
// dipakai sebagai audience (aud) sesi login platform admin di UI console, agar
// admin tidak perlu memilih solution produk. Idempotent: solution yang sudah
// ada tidak diubah.
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
	_, err := db.ExecContext(ctx, `
		INSERT INTO solutions (code, name, status)
		VALUES ($1, $2, 'active')
		ON CONFLICT (code) DO NOTHING`, s.Code, name)
	if err != nil {
		return err
	}
	return nil
}
