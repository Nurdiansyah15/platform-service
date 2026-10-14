package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/signingkey/constant"
	"platform-service/internal/domain/signingkey/entity"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func TestPostgresSigningKeyRepository(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		// Kosongkan key yang mungkin dibuat test lain (dalam tx, otomatis rollback).
		_, err := persistence.TxFromContext(ctx).ExecContext(ctx, `DELETE FROM signing_keys`)
		require.NoError(t, err)

		repo := persistence.NewPostgresSigningKeyRepository(testDB)

		_, err = repo.FindActive(ctx)
		testhelper.AssertDomainError(t, err, constant.CodeSigningKeyNotFound)

		first, err := entity.NewSigningKey("pem-1", "mem:1", time.Now())
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, first))

		second, err := entity.NewSigningKey("pem-2", "mem:2", time.Now())
		require.NoError(t, err)
		// Error di dalam transaksi Postgres membatalkan tx; savepoint menjaga tx test tetap hidup.
		tx := persistence.TxFromContext(ctx)
		_, err = tx.ExecContext(ctx, `SAVEPOINT dup_active`)
		require.NoError(t, err)
		testhelper.AssertDomainError(t, repo.Save(ctx, second), constant.CodeSigningKeyActiveExists)
		_, err = tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT dup_active`)
		require.NoError(t, err)

		// Rotasi: key lama → retiring, key baru → active.
		require.NoError(t, first.StartRetiring())
		require.NoError(t, repo.Save(ctx, first))
		require.NoError(t, repo.Save(ctx, second))

		active, err := repo.FindActive(ctx)
		require.NoError(t, err)
		assert.Equal(t, second.KID, active.KID)

		published, err := repo.ListPublished(ctx)
		require.NoError(t, err)
		assert.Len(t, published, 2)

		require.NoError(t, first.Retire(time.Now()))
		require.NoError(t, repo.Save(ctx, first))
		published, err = repo.ListPublished(ctx)
		require.NoError(t, err)
		require.Len(t, published, 1)
		assert.Equal(t, second.KID, published[0].KID)

		got, err := repo.FindByKID(ctx, first.KID)
		require.NoError(t, err)
		assert.False(t, got.Status.IsPublished())
		assert.NotNil(t, got.RetiredAt)
	})
}
