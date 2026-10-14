package testhelper

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// UniqueCode menghasilkan code unik huruf kecil (aman untuk code solution/plan/dll).
func UniqueCode(prefix string) string {
	return prefix + "-" + uuid.NewString()[:8]
}

// MustSeedSolution menyisipkan solution langsung ke DB dan mengembalikan ID-nya.
func MustSeedSolution(t *testing.T, db *sql.DB, code, status string) string {
	t.Helper()
	var id string
	err := db.QueryRowContext(context.Background(),
		`INSERT INTO solutions (code, name, status) VALUES ($1, $2, $3) RETURNING id`,
		code, "Solution "+code, status,
	).Scan(&id)
	require.NoError(t, err, "MustSeedSolution")
	return id
}

// MustSeedSigningKey membuat key pair di key store server lalu menyisipkan baris
// signing_keys dengan status tertentu (mis. "retiring" / "retired"). Mengembalikan kid.
func MustSeedSigningKey(t *testing.T, srv *TestServer, status string) string {
	t.Helper()
	ref, pubPEM, err := srv.KeyStore.Generate(context.Background())
	require.NoError(t, err)
	kid := "test-" + uuid.NewString()[:8]
	_, err = srv.DB.ExecContext(context.Background(),
		`INSERT INTO signing_keys (kid, public_key, private_key_ref, status) VALUES ($1, $2, $3, $4)`,
		kid, pubPEM, ref, status)
	require.NoError(t, err, "MustSeedSigningKey")
	return kid
}
