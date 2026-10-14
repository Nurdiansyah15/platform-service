// Package seeders berisi data awal (mis. platform admin, solution contoh).
// Seeder wajib idempotent karena bisa dijalankan berulang.
package seeders

import (
	"context"
	"database/sql"
)

type Seeder interface {
	Name() string
	Run(ctx context.Context, db *sql.DB) error
}

// All mengembalikan semua seeder sesuai urutan eksekusi. Seeder pertama
// (platform admin) ditambahkan di Fase 3.
func All() []Seeder {
	return nil
}
