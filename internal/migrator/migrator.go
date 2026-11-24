// Package migrator menjalankan migrasi SQL dan mencatat SETIAP file yang sudah
// diterapkan di tabel riwayat (bukan hanya versi terakhir seperti golang-migrate).
//
// Akibatnya migrasi dari branch lain yang timestamp-nya lebih tua dari migrasi
// terakhir yang sudah jalan tetap terdeteksi sebagai pending dan dijalankan saat
// `up` berikutnya — tidak ada yang terlewat diam-diam.
//
// Format file: <version>_<name>.up.sql dan <version>_<name>.down.sql, version
// berupa angka (timestamp UTC yyyymmddhhmmss dari `make dev-migrate-create`).
// Setiap file dijalankan dalam transaksi sendiri bersama pencatatan riwayatnya,
// jadi migrasi yang gagal tidak meninggalkan state "dirty".
package migrator

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"time"
)

// HistoryTable menyimpan satu baris per file migrasi yang sudah diterapkan.
const HistoryTable = "migrations"

// advisoryLockKey mencegah dua proses migrasi berjalan bersamaan pada DB yang sama.
const advisoryLockKey int64 = 7_382_910_455_120_001

var fileRe = regexp.MustCompile(`^(\d+)_([a-z0-9_]+)\.(up|down)\.sql$`)

// Migration adalah pasangan file up/down dengan versi yang sama.
type Migration struct {
	Version  int64
	Name     string
	UpSQL    string
	DownSQL  string
	Checksum string // sha256 isi file up
}

// ID mengembalikan nama dasar file tanpa suffix, mis. "20261014083155_create_companies".
func (m Migration) ID() string { return fmt.Sprintf("%d_%s", m.Version, m.Name) }

// Applied adalah baris riwayat di DB.
type Applied struct {
	Version   int64
	Name      string
	Checksum  string
	AppliedAt time.Time
}

// State status satu migrasi pada laporan Status.
type State string

const (
	StateApplied State = "applied"
	StatePending State = "pending"
	StateChanged State = "changed" // sudah diterapkan, tapi isi file up berubah sejak itu
	StateMissing State = "missing" // tercatat di DB, file tidak ada (mis. dari branch lain)
)

// StatusRow satu baris laporan Status, urut berdasarkan versi.
type StatusRow struct {
	Version   int64
	Name      string
	State     State
	AppliedAt *time.Time
}

// Logger dipakai untuk melaporkan progres; *log.Logger memenuhinya.
type Logger interface {
	Printf(format string, v ...any)
}

type nopLogger struct{}

func (nopLogger) Printf(string, ...any) {}

// Option mengatur Migrator.
type Option func(*Migrator)

// WithLogger memasang logger progres.
func WithLogger(l Logger) Option { return func(m *Migrator) { m.log = l } }

// WithSchema menjalankan migrasi di schema tertentu (search_path). Default:
// schema bawaan koneksi (biasanya public).
func WithSchema(schema string) Option { return func(m *Migrator) { m.schema = schema } }

// Migrator menjalankan migrasi dari fs.FS ke database.
type Migrator struct {
	db         *sql.DB
	migrations []Migration
	log        Logger
	schema     string
}

// New membaca & memvalidasi semua file migrasi di root fsys.
func New(db *sql.DB, fsys fs.FS, opts ...Option) (*Migrator, error) {
	migs, err := Load(fsys)
	if err != nil {
		return nil, err
	}
	m := &Migrator{db: db, migrations: migs, log: nopLogger{}}
	for _, o := range opts {
		o(m)
	}
	return m, nil
}

// Load membaca file migrasi di root fsys, memvalidasi nama, pasangan up/down,
// dan keunikan versi, lalu mengurutkannya berdasarkan versi.
func Load(fsys fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("migrator: baca direktori migrasi: %w", err)
	}

	byVersion := map[int64]*Migration{}
	for _, e := range entries {
		if e.IsDir() || !isSQL(e.Name()) {
			continue
		}
		match := fileRe.FindStringSubmatch(e.Name())
		if match == nil {
			return nil, fmt.Errorf("migrator: nama file tidak valid %q (format: <version>_<snake_name>.up|down.sql)", e.Name())
		}
		version, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("migrator: versi tidak valid pada %q: %w", e.Name(), err)
		}
		name, direction := match[2], match[3]

		body, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return nil, fmt.Errorf("migrator: baca %q: %w", e.Name(), err)
		}

		mig, ok := byVersion[version]
		if !ok {
			mig = &Migration{Version: version, Name: name}
			byVersion[version] = mig
		} else if mig.Name != name {
			return nil, fmt.Errorf("migrator: versi %d dipakai dua migrasi berbeda (%q dan %q)", version, mig.Name, name)
		}

		if direction == "up" {
			if mig.UpSQL != "" {
				return nil, fmt.Errorf("migrator: file up ganda untuk versi %d", version)
			}
			mig.UpSQL = string(body)
			sum := sha256.Sum256(body)
			mig.Checksum = hex.EncodeToString(sum[:])
		} else {
			if mig.DownSQL != "" {
				return nil, fmt.Errorf("migrator: file down ganda untuk versi %d", version)
			}
			mig.DownSQL = string(body)
		}
	}

	out := make([]Migration, 0, len(byVersion))
	for _, mig := range byVersion {
		if mig.Checksum == "" {
			return nil, fmt.Errorf("migrator: %s tidak punya file .up.sql", mig.ID())
		}
		out = append(out, *mig)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func isSQL(name string) bool {
	return len(name) > 4 && name[len(name)-4:] == ".sql"
}

// Migrations mengembalikan migrasi yang dimuat, urut versi.
func (m *Migrator) Migrations() []Migration { return m.migrations }

// Up menjalankan semua migrasi yang belum tercatat di riwayat, urut versi —
// termasuk yang versinya lebih tua dari migrasi terakhir yang sudah diterapkan.
// Mengembalikan daftar migrasi yang dijalankan.
func (m *Migrator) Up(ctx context.Context) ([]Migration, error) {
	var ran []Migration
	err := m.withConn(ctx, func(conn *sql.Conn) error {
		applied, err := loadApplied(ctx, conn)
		if err != nil {
			return err
		}
		for _, mig := range m.migrations {
			if a, ok := applied[mig.Version]; ok {
				if a.Checksum != mig.Checksum {
					m.log.Printf("PERINGATAN: isi %s berubah sejak diterapkan (tidak dijalankan ulang)", mig.ID())
				}
				continue
			}
			m.log.Printf("up   %s", mig.ID())
			if err := m.applyUp(ctx, conn, mig); err != nil {
				return err
			}
			ran = append(ran, mig)
		}
		return nil
	})
	return ran, err
}

// Down membatalkan `steps` migrasi terakhir berdasarkan URUTAN DITERAPKAN (bukan
// versi), sehingga migrasi branch lain yang baru saja dijalankan ikut terbalik
// lebih dulu. Mengembalikan daftar migrasi yang dibatalkan.
func (m *Migrator) Down(ctx context.Context, steps int) ([]Migration, error) {
	if steps < 1 {
		return nil, fmt.Errorf("migrator: jumlah langkah down minimal 1")
	}
	files := make(map[int64]Migration, len(m.migrations))
	for _, mig := range m.migrations {
		files[mig.Version] = mig
	}

	var reverted []Migration
	err := m.withConn(ctx, func(conn *sql.Conn) error {
		rows, err := conn.QueryContext(ctx,
			`SELECT version, name FROM `+HistoryTable+` ORDER BY id DESC LIMIT $1`, steps)
		if err != nil {
			return fmt.Errorf("migrator: baca riwayat: %w", err)
		}
		type rec struct {
			version int64
			name    string
		}
		var targets []rec
		for rows.Next() {
			var r rec
			if err := rows.Scan(&r.version, &r.name); err != nil {
				_ = rows.Close()
				return fmt.Errorf("migrator: baca riwayat: %w", err)
			}
			targets = append(targets, r)
		}
		_ = rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("migrator: baca riwayat: %w", err)
		}

		for _, t := range targets {
			mig, ok := files[t.version]
			if !ok {
				return fmt.Errorf("migrator: file untuk %d_%s tidak ditemukan — checkout branch yang memilikinya sebelum down", t.version, t.name)
			}
			if mig.DownSQL == "" {
				return fmt.Errorf("migrator: %s tidak punya file .down.sql", mig.ID())
			}
			m.log.Printf("down %s", mig.ID())
			if err := m.applyDown(ctx, conn, mig); err != nil {
				return err
			}
			reverted = append(reverted, mig)
		}
		return nil
	})
	return reverted, err
}

// Fresh menghapus seluruh isi schema (DROP SCHEMA ... CASCADE lalu dibuat ulang)
// kemudian menjalankan Up. Hanya untuk development/test.
func (m *Migrator) Fresh(ctx context.Context) ([]Migration, error) {
	err := m.withConn(ctx, func(conn *sql.Conn) error {
		var schema string
		if err := conn.QueryRowContext(ctx, `SELECT current_schema()`).Scan(&schema); err != nil {
			return fmt.Errorf("migrator: baca schema aktif: %w", err)
		}
		q := quoteIdent(schema)
		m.log.Printf("drop schema %s", schema)
		if _, err := conn.ExecContext(ctx, `DROP SCHEMA `+q+` CASCADE; CREATE SCHEMA `+q); err != nil {
			return fmt.Errorf("migrator: drop schema %s: %w", schema, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return m.Up(ctx)
}

// Status membandingkan file migrasi dengan riwayat di DB.
func (m *Migrator) Status(ctx context.Context) ([]StatusRow, error) {
	var out []StatusRow
	err := m.withConn(ctx, func(conn *sql.Conn) error {
		applied, err := loadApplied(ctx, conn)
		if err != nil {
			return err
		}
		seen := make(map[int64]bool, len(m.migrations))
		for _, mig := range m.migrations {
			seen[mig.Version] = true
			row := StatusRow{Version: mig.Version, Name: mig.Name, State: StatePending}
			if a, ok := applied[mig.Version]; ok {
				at := a.AppliedAt
				row.AppliedAt = &at
				row.State = StateApplied
				if a.Checksum != mig.Checksum {
					row.State = StateChanged
				}
			}
			out = append(out, row)
		}
		for v, a := range applied {
			if seen[v] {
				continue
			}
			at := a.AppliedAt
			out = append(out, StatusRow{Version: v, Name: a.Name, State: StateMissing, AppliedAt: &at})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
		return nil
	})
	return out, err
}

// withConn memakai satu koneksi untuk seluruh operasi: set search_path (bila
// diminta), ambil advisory lock, pastikan tabel riwayat ada, lalu jalankan fn.
func (m *Migrator) withConn(ctx context.Context, fn func(*sql.Conn) error) error {
	conn, err := m.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("migrator: ambil koneksi: %w", err)
	}
	defer conn.Close()

	if m.schema != "" {
		q := quoteIdent(m.schema)
		if _, err := conn.ExecContext(ctx, `CREATE SCHEMA IF NOT EXISTS `+q+`; SET search_path TO `+q); err != nil {
			return fmt.Errorf("migrator: set schema %s: %w", m.schema, err)
		}
		// Koneksi dikembalikan ke pool; kembalikan search_path agar tidak bocor.
		defer func() { _, _ = conn.ExecContext(context.Background(), `RESET search_path`) }()
	}

	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, advisoryLockKey); err != nil {
		return fmt.Errorf("migrator: ambil advisory lock: %w", err)
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, advisoryLockKey)
	}()

	if err := m.ensureHistory(ctx, conn); err != nil {
		return err
	}
	return fn(conn)
}

func (m *Migrator) ensureHistory(ctx context.Context, conn *sql.Conn) error {
	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS `+HistoryTable+` (
		id         BIGSERIAL PRIMARY KEY,
		version    BIGINT      NOT NULL UNIQUE,
		name       TEXT        NOT NULL,
		checksum   TEXT        NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("migrator: buat tabel riwayat: %w", err)
	}
	return nil
}

func loadApplied(ctx context.Context, conn *sql.Conn) (map[int64]Applied, error) {
	rows, err := conn.QueryContext(ctx, `SELECT version, name, checksum, applied_at FROM `+HistoryTable)
	if err != nil {
		return nil, fmt.Errorf("migrator: baca riwayat: %w", err)
	}
	defer rows.Close()
	out := map[int64]Applied{}
	for rows.Next() {
		var a Applied
		if err := rows.Scan(&a.Version, &a.Name, &a.Checksum, &a.AppliedAt); err != nil {
			return nil, fmt.Errorf("migrator: baca riwayat: %w", err)
		}
		out[a.Version] = a
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrator: baca riwayat: %w", err)
	}
	return out, nil
}

func (m *Migrator) applyUp(ctx context.Context, conn *sql.Conn, mig Migration) error {
	return inTx(ctx, conn, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, mig.UpSQL); err != nil {
			return fmt.Errorf("migrator: up %s: %w", mig.ID(), err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO `+HistoryTable+` (version, name, checksum) VALUES ($1, $2, $3)`,
			mig.Version, mig.Name, mig.Checksum); err != nil {
			return fmt.Errorf("migrator: catat %s: %w", mig.ID(), err)
		}
		return nil
	})
}

func (m *Migrator) applyDown(ctx context.Context, conn *sql.Conn, mig Migration) error {
	return inTx(ctx, conn, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, mig.DownSQL); err != nil {
			return fmt.Errorf("migrator: down %s: %w", mig.ID(), err)
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM `+HistoryTable+` WHERE version = $1`, mig.Version); err != nil {
			return fmt.Errorf("migrator: hapus riwayat %s: %w", mig.ID(), err)
		}
		return nil
	})
}

func inTx(ctx context.Context, conn *sql.Conn, fn func(*sql.Tx) error) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrator: mulai transaksi: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrator: commit: %w", err)
	}
	return nil
}

func quoteIdent(s string) string {
	out := make([]byte, 0, len(s)+2)
	out = append(out, '"')
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			out = append(out, '"')
		}
		out = append(out, s[i])
	}
	return string(append(out, '"'))
}
