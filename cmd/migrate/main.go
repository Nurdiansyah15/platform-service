// Command migrate menjalankan migrasi database dari file SQL yang di-embed
// (internal/migrations). Pemakaian: migrate [up|down|fresh|version|force VERSION]
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"platform-service/internal/config"
	"platform-service/internal/migrations"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: migrate [up|down|fresh|version|force VERSION]")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("gagal load config: %v", err)
	}

	m := newMigrator(cfg)
	defer m.Close()

	switch cmd := os.Args[1]; cmd {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("migrate up gagal: %v", err)
		}
		log.Println("Migrasi UP berhasil")

	case "down":
		if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("migrate down gagal: %v", err)
		}
		log.Println("Migrasi DOWN (mundur 1 langkah) berhasil")

	case "fresh":
		if err := m.Drop(); err != nil {
			log.Fatalf("migrate fresh gagal saat drop: %v", err)
		}
		// Drop menghapus tabel schema_migrations; migrator baru diperlukan untuk Up.
		fresh := newMigrator(cfg)
		defer fresh.Close()
		if err := fresh.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("migrate fresh gagal saat up: %v", err)
		}
		log.Println("Migrasi FRESH berhasil")

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("gagal cek versi: %v", err)
		}
		fmt.Printf("Versi: %d (dirty: %v)\n", version, dirty)

	case "force":
		if len(os.Args) < 3 {
			fmt.Println("Usage: migrate force VERSION")
			os.Exit(1)
		}
		version, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("VERSION tidak valid: %v", err)
		}
		if err := m.Force(version); err != nil {
			log.Fatalf("migrate force gagal: %v", err)
		}
		log.Printf("Force ke versi %d berhasil", version)

	default:
		fmt.Printf("Command tidak dikenal: %s\n", cmd)
		os.Exit(1)
	}
}

func newMigrator(cfg *config.Config) *migrate.Migrate {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		log.Fatalf("gagal membaca migrasi: %v", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, migrateURL(cfg.Database.DSN()))
	if err != nil {
		log.Fatalf("gagal init migrator: %v", err)
	}
	return m
}

// migrateURL mengganti scheme ke "pgx5" (driver golang-migrate berbasis pgx/v5).
func migrateURL(dsn string) string {
	const prefix = "postgres://"
	if len(dsn) > len(prefix) && dsn[:len(prefix)] == prefix {
		return "pgx5://" + dsn[len(prefix):]
	}
	return dsn
}
