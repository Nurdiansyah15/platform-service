package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/solutionclient/constant"
	"platform-service/internal/domain/solutionclient/entity"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func TestPostgresSolutionClientRepository(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		var solutionID string
		require.NoError(t, persistence.TxFromContext(ctx).QueryRowContext(ctx,
			`INSERT INTO solutions (code, name) VALUES ($1, 'S') RETURNING id`, testhelper.UniqueCode("cli")).Scan(&solutionID))

		repo := persistence.NewPostgresSolutionClientRepository(testDB)
		c, err := entity.NewSolutionClient(entity.NewSolutionClientParams{
			SolutionID: solutionID, Name: "Backend", ClientID: "cli_" + testhelper.UniqueCode("x"),
			ClientSecretHash: "hash", Scopes: []string{"usage:write", "authz:check"},
		}, time.Now())
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, c))

		got, err := repo.FindByClientID(ctx, c.ClientID)
		require.NoError(t, err)
		assert.Equal(t, c.ID, got.ID)
		assert.Equal(t, []string{"authz:check", "usage:write"}, got.Scopes.Values(), "TEXT[] round trip")
		assert.Nil(t, got.LastUsedAt)

		require.NoError(t, repo.TouchLastUsed(ctx, c.ID, time.Now()))
		got.Revoke(time.Now())
		require.NoError(t, repo.Save(ctx, got))

		again, err := repo.FindByID(ctx, c.ID)
		require.NoError(t, err)
		assert.False(t, again.IsActive())
		assert.NotNil(t, again.RevokedAt)
		assert.NotNil(t, again.LastUsedAt)

		_, err = repo.FindByClientID(ctx, "cli_tidak_ada")
		testhelper.AssertDomainError(t, err, constant.CodeClientNotFound)
	})
}
