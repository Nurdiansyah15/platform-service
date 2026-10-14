// Package seeders berisi data awal (mis. platform admin). Seeder wajib
// idempotent karena bisa dijalankan berulang.
package seeders

import (
	"context"
	"database/sql"
)

type Seeder interface {
	Name() string
	Run(ctx context.Context, db *sql.DB) error
}

// Config berisi nilai yang dibutuhkan seeder (dibaca cmd/seeder dari env).
type Config struct {
	PlatformAdminEmail    string
	PlatformAdminName     string
	PlatformAdminPassword string
}

// All mengembalikan semua seeder sesuai urutan eksekusi.
func All(cfg Config) []Seeder {
	return []Seeder{
		PlatformAdminSeeder{Email: cfg.PlatformAdminEmail, AdminName: cfg.PlatformAdminName, Password: cfg.PlatformAdminPassword},
	}
}
