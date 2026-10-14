package persistence_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/solution/constant"
	"platform-service/internal/domain/solution/entity"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func newTestSolution(t *testing.T) *entity.Solution {
	t.Helper()
	s, err := entity.NewSolution(entity.NewSolutionParams{Code: testhelper.UniqueCode("repo"), Name: "Repo Test"})
	require.NoError(t, err)
	return s
}

func TestPostgresSolutionRepository_SaveAndFind(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		repo := persistence.NewPostgresSolutionRepository(testDB)
		s := newTestSolution(t)
		require.NoError(t, repo.Save(ctx, s))

		got, err := repo.FindByID(ctx, s.ID)
		require.NoError(t, err)
		assert.Equal(t, s.Code.Value(), got.Code.Value())
		assert.Equal(t, "Repo Test", got.Name)
		assert.True(t, got.IsActive())
	})
}

func TestPostgresSolutionRepository_Save_DuplicateCode(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		repo := persistence.NewPostgresSolutionRepository(testDB)
		first := newTestSolution(t)
		require.NoError(t, repo.Save(ctx, first))

		dup, err := entity.NewSolution(entity.NewSolutionParams{Code: first.Code.Value(), Name: "Dup"})
		require.NoError(t, err)
		testhelper.AssertDomainError(t, repo.Save(ctx, dup), constant.CodeSolutionCodeDuplicate)
	})
}

func TestPostgresSolutionRepository_FindByID_NotFound(t *testing.T) {
	_, err := persistence.NewPostgresSolutionRepository(testDB).FindByID(context.Background(), uuid.NewString())
	testhelper.AssertDomainError(t, err, constant.CodeSolutionNotFound)
}
