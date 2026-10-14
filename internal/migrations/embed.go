// Package migrations menyimpan file migrasi SQL yang di-embed ke binary
// (dipakai cmd/migrate dan testhelper). Buat file baru dengan
// `make migrate-create NAME=...` — jangan menulis nomor versi manual.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
