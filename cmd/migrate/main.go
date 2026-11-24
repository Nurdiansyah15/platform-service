// Command migrate menjalankan migrasi database dari file SQL yang di-embed
// (internal/migrations) memakai internal/migrator: setiap file yang diterapkan
// dicatat di tabel migrations, jadi migrasi dari branch lain yang
// versinya lebih tua tetap dijalankan.
//
// Pemakaian: migrate [up | down [N] | fresh | status]
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"text/tabwriter"

	_ "github.com/jackc/pgx/v5/stdlib"

	"platform-service/internal/config"
	"platform-service/internal/migrations"
	"platform-service/internal/migrator"
)

const usage = "Usage: migrate [up | down [N] | fresh | status]"

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("gagal load config: %v", err)
	}

	db, err := sql.Open("pgx", cfg.Database.DSN())
	if err != nil {
		log.Fatalf("gagal membuka koneksi database: %v", err)
	}
	defer db.Close()

	m, err := migrator.New(db, migrations.FS, migrator.WithLogger(log.Default()))
	if err != nil {
		log.Fatalf("gagal membaca migrasi: %v", err)
	}

	ctx := context.Background()
	switch cmd := os.Args[1]; cmd {
	case "up":
		ran, err := m.Up(ctx)
		if err != nil {
			log.Fatalf("migrate up gagal: %v", err)
		}
		log.Printf("Migrasi UP berhasil (%d file dijalankan)", len(ran))

	case "down":
		steps := 1
		if len(os.Args) >= 3 {
			steps, err = strconv.Atoi(os.Args[2])
			if err != nil || steps < 1 {
				log.Fatalf("N tidak valid: %q", os.Args[2])
			}
		}
		reverted, err := m.Down(ctx, steps)
		if err != nil {
			log.Fatalf("migrate down gagal: %v", err)
		}
		log.Printf("Migrasi DOWN berhasil (%d file dibatalkan)", len(reverted))

	case "fresh":
		ran, err := m.Fresh(ctx)
		if err != nil {
			log.Fatalf("migrate fresh gagal: %v", err)
		}
		log.Printf("Migrasi FRESH berhasil (%d file dijalankan)", len(ran))

	case "status", "version":
		rows, err := m.Status(ctx)
		if err != nil {
			log.Fatalf("gagal cek status: %v", err)
		}
		printStatus(rows)

	default:
		fmt.Printf("Command tidak dikenal: %s\n%s\n", cmd, usage)
		os.Exit(1)
	}
}

func printStatus(rows []migrator.StatusRow) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "STATUS\tMIGRASI\tDITERAPKAN")
	counts := map[migrator.State]int{}
	for _, r := range rows {
		counts[r.State]++
		at := "-"
		if r.AppliedAt != nil {
			at = r.AppliedAt.Local().Format("2006-01-02 15:04:05")
		}
		fmt.Fprintf(w, "%s\t%d_%s\t%s\n", r.State, r.Version, r.Name, at)
	}
	_ = w.Flush()
	fmt.Printf("\napplied: %d, pending: %d, changed: %d, missing: %d\n",
		counts[migrator.StateApplied], counts[migrator.StatePending],
		counts[migrator.StateChanged], counts[migrator.StateMissing])
	if counts[migrator.StateChanged] > 0 {
		fmt.Println("changed = isi file .up.sql berubah setelah diterapkan (tidak dijalankan ulang otomatis).")
	}
	if counts[migrator.StateMissing] > 0 {
		fmt.Println("missing = tercatat di DB tapi file tidak ada (kemungkinan dari branch lain).")
	}
}
