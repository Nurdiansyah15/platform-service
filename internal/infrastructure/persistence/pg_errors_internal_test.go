package persistence

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestPgErrorHelpers(t *testing.T) {
	wrap := func(code, constraint string) error {
		return fmt.Errorf("repo: %w", &pgconn.PgError{Code: code, ConstraintName: constraint})
	}

	unique := wrap(pgUniqueViolation, "uq_companies_slug")
	assert.True(t, isUniqueViolation(unique))
	assert.False(t, isForeignKeyViolation(unique))
	assert.Equal(t, "uq_companies_slug", constraintName(unique))

	assert.True(t, isForeignKeyViolation(wrap(pgForeignKeyViolation, "")))
	assert.True(t, isCheckViolation(wrap(pgCheckViolation, "")))

	plain := errors.New("bukan error postgres")
	assert.False(t, isUniqueViolation(plain))
	assert.Empty(t, constraintName(plain))
}
