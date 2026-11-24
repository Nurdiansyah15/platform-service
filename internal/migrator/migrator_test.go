package migrator_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/migrator"
	"platform-service/internal/testhelper"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	var cleanup func()
	testDB, cleanup = testhelper.MustStartTestDB()
	code := m.Run()
	cleanup()
	os.Exit(code)
}

var schemaSeq atomic.Int64

// newSchema membuat nama schema unik; migrator membuatnya saat pertama dipakai.
func newSchema(t *testing.T) string {
	t.Helper()
	name := fmt.Sprintf("mig_test_%d", schemaSeq.Add(1))
	t.Cleanup(func() {
		_, _ = testDB.Exec(`DROP SCHEMA IF EXISTS ` + name + ` CASCADE`)
	})
	return name
}

func file(sql string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(sql)} }

func baseFS() fstest.MapFS {
	return fstest.MapFS{
		"20260101000000_create_a.up.sql":   file(`CREATE TABLE a (id INT);`),
		"20260101000000_create_a.down.sql": file(`DROP TABLE a;`),
		"20260103000000_create_c.up.sql":   file(`CREATE TABLE c (id INT); INSERT INTO c VALUES (1);`),
		"20260103000000_create_c.down.sql": file(`DROP TABLE c;`),
	}
}

func mustNew(t *testing.T, fsys fstest.MapFS, schema string) *migrator.Migrator {
	t.Helper()
	m, err := migrator.New(testDB, fsys, migrator.WithSchema(schema))
	require.NoError(t, err)
	return m
}

func tableExists(t *testing.T, schema, table string) bool {
	t.Helper()
	var ok bool
	require.NoError(t, testDB.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, schema+"."+table).Scan(&ok))
	return ok
}

func appliedVersions(t *testing.T, schema string) []int64 {
	t.Helper()
	rows, err := testDB.Query(`SELECT version FROM ` + schema + `.migrations ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var v int64
		require.NoError(t, rows.Scan(&v))
		out = append(out, v)
	}
	require.NoError(t, rows.Err())
	return out
}

func ids(migs []migrator.Migration) []string {
	out := make([]string, len(migs))
	for i, m := range migs {
		out[i] = m.ID()
	}
	return out
}

// ── Load ─────────────────────────────────────────────────────────────────────

func TestLoad_SortsByVersion(t *testing.T) {
	migs, err := migrator.Load(baseFS())
	require.NoError(t, err)
	assert.Equal(t, []string{"20260101000000_create_a", "20260103000000_create_c"}, ids(migs))
	assert.NotEmpty(t, migs[0].Checksum)
}

func TestLoad_InvalidName(t *testing.T) {
	_, err := migrator.Load(fstest.MapFS{"create_a.up.sql": file(`SELECT 1`)})
	assert.ErrorContains(t, err, "nama file tidak valid")
}

func TestLoad_MissingUp(t *testing.T) {
	_, err := migrator.Load(fstest.MapFS{"20260101000000_a.down.sql": file(`SELECT 1`)})
	assert.ErrorContains(t, err, "tidak punya file .up.sql")
}

func TestLoad_DuplicateVersion(t *testing.T) {
	_, err := migrator.Load(fstest.MapFS{
		"20260101000000_a.up.sql": file(`SELECT 1`),
		"20260101000000_b.up.sql": file(`SELECT 1`),
	})
	assert.ErrorContains(t, err, "dipakai dua migrasi berbeda")
}

// ── Up ───────────────────────────────────────────────────────────────────────

func TestUp_AppliesAllThenNoop(t *testing.T) {
	ctx := context.Background()
	schema := newSchema(t)
	m := mustNew(t, baseFS(), schema)

	ran, err := m.Up(ctx)
	require.NoError(t, err)
	assert.Len(t, ran, 2)
	assert.True(t, tableExists(t, schema, "a"))
	assert.True(t, tableExists(t, schema, "c"))

	ran, err = m.Up(ctx)
	require.NoError(t, err)
	assert.Empty(t, ran)
}

// Inti perubahan: migrasi dari branch lain yang versinya LEBIH TUA dari migrasi
// terakhir yang sudah jalan tetap dijalankan.
func TestUp_RunsOlderMigrationAddedLater(t *testing.T) {
	ctx := context.Background()
	schema := newSchema(t)

	_, err := mustNew(t, baseFS(), schema).Up(ctx)
	require.NoError(t, err)

	fsys := baseFS()
	fsys["20260102000000_create_b.up.sql"] = file(`CREATE TABLE b (id INT);`)
	fsys["20260102000000_create_b.down.sql"] = file(`DROP TABLE b;`)

	ran, err := mustNew(t, fsys, schema).Up(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"20260102000000_create_b"}, ids(ran))
	assert.True(t, tableExists(t, schema, "b"))
	assert.Equal(t, []int64{20260101000000, 20260103000000, 20260102000000}, appliedVersions(t, schema))
}

func TestUp_FailureRollsBackFile(t *testing.T) {
	ctx := context.Background()
	schema := newSchema(t)
	fsys := baseFS()
	fsys["20260104000000_broken.up.sql"] = file(`CREATE TABLE d (id INT); SELECT * FROM tidak_ada;`)
	fsys["20260104000000_broken.down.sql"] = file(`DROP TABLE d;`)

	_, err := mustNew(t, fsys, schema).Up(ctx)
	require.Error(t, err)
	assert.ErrorContains(t, err, "20260104000000_broken")
	assert.False(t, tableExists(t, schema, "d"), "statement sebelum error ikut dibatalkan")
	assert.Equal(t, []int64{20260101000000, 20260103000000}, appliedVersions(t, schema))
}

// ── Down ─────────────────────────────────────────────────────────────────────

func TestDown_RevertsInAppliedOrder(t *testing.T) {
	ctx := context.Background()
	schema := newSchema(t)
	_, err := mustNew(t, baseFS(), schema).Up(ctx)
	require.NoError(t, err)

	fsys := baseFS()
	fsys["20260102000000_create_b.up.sql"] = file(`CREATE TABLE b (id INT);`)
	fsys["20260102000000_create_b.down.sql"] = file(`DROP TABLE b;`)
	m := mustNew(t, fsys, schema)
	_, err = m.Up(ctx)
	require.NoError(t, err)

	reverted, err := m.Down(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, []string{"20260102000000_create_b"}, ids(reverted), "yang terakhir diterapkan, bukan versi tertinggi")
	assert.False(t, tableExists(t, schema, "b"))
	assert.True(t, tableExists(t, schema, "c"))

	reverted, err = m.Down(ctx, 5)
	require.NoError(t, err)
	assert.Equal(t, []string{"20260103000000_create_c", "20260101000000_create_a"}, ids(reverted))
	assert.Empty(t, appliedVersions(t, schema))
}

func TestDown_MissingFile(t *testing.T) {
	ctx := context.Background()
	schema := newSchema(t)
	_, err := mustNew(t, baseFS(), schema).Up(ctx)
	require.NoError(t, err)

	fsys := baseFS()
	delete(fsys, "20260103000000_create_c.up.sql")
	delete(fsys, "20260103000000_create_c.down.sql")
	_, err = mustNew(t, fsys, schema).Down(ctx, 1)
	assert.ErrorContains(t, err, "tidak ditemukan")
}

// ── Status ───────────────────────────────────────────────────────────────────

func TestStatus_States(t *testing.T) {
	ctx := context.Background()
	schema := newSchema(t)
	_, err := mustNew(t, baseFS(), schema).Up(ctx)
	require.NoError(t, err)

	fsys := baseFS()
	fsys["20260101000000_create_a.up.sql"] = file(`CREATE TABLE a (id INT, x INT);`) // diubah
	delete(fsys, "20260103000000_create_c.up.sql")                                   // hilang
	delete(fsys, "20260103000000_create_c.down.sql")
	fsys["20260105000000_create_e.up.sql"] = file(`CREATE TABLE e (id INT);`) // baru

	rows, err := mustNew(t, fsys, schema).Status(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, migrator.StateChanged, rows[0].State)
	assert.Equal(t, migrator.StateMissing, rows[1].State)
	assert.Equal(t, migrator.StatePending, rows[2].State)
	assert.Nil(t, rows[2].AppliedAt)
}

// ── Fresh ────────────────────────────────────────────────────────────────────

func TestFresh_DropsAndReapplies(t *testing.T) {
	ctx := context.Background()
	schema := newSchema(t)
	m := mustNew(t, baseFS(), schema)
	_, err := m.Up(ctx)
	require.NoError(t, err)
	_, err = testDB.Exec(`CREATE TABLE ` + schema + `.leftover (id INT)`)
	require.NoError(t, err)

	ran, err := m.Fresh(ctx)
	require.NoError(t, err)
	assert.Len(t, ran, 2)
	assert.False(t, tableExists(t, schema, "leftover"))
	var n int
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM `+schema+`.c`).Scan(&n))
	assert.Equal(t, 1, n)
}
