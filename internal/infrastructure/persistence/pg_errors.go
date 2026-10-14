package persistence

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// SQLSTATE yang dipetakan repository ke domain error (lihat CLAUDE.md §6).
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgCheckViolation      = "23514"
)

// pgError mengembalikan *pgconn.PgError bila err berasal dari Postgres.
func pgError(err error) (*pgconn.PgError, bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr, true
	}
	return nil, false
}

func isUniqueViolation(err error) bool {
	pgErr, ok := pgError(err)
	return ok && pgErr.Code == pgUniqueViolation
}

func isForeignKeyViolation(err error) bool {
	pgErr, ok := pgError(err)
	return ok && pgErr.Code == pgForeignKeyViolation
}

func isCheckViolation(err error) bool {
	pgErr, ok := pgError(err)
	return ok && pgErr.Code == pgCheckViolation
}

// constraintName mengembalikan nama constraint yang dilanggar (kosong bila bukan error Postgres).
// Dipakai saat satu tabel punya beberapa unique/FK dan error domain-nya berbeda.
func constraintName(err error) string {
	if pgErr, ok := pgError(err); ok {
		return pgErr.ConstraintName
	}
	return ""
}
