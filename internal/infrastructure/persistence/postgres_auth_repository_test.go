package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sessionconstant "platform-service/internal/domain/authsession/constant"
	sessionentity "platform-service/internal/domain/authsession/entity"
	userconstant "platform-service/internal/domain/user/constant"
	userentity "platform-service/internal/domain/user/entity"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func mustSaveUser(t *testing.T, ctx context.Context, email string) *userentity.User {
	t.Helper()
	u, err := userentity.NewUser(userentity.NewUserParams{Email: email, Name: "Repo"}, time.Now())
	require.NoError(t, err)
	require.NoError(t, persistence.NewPostgresUserRepository(testDB).Save(ctx, u))
	return u
}

func TestPostgresUserRepository(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		repo := persistence.NewPostgresUserRepository(testDB)
		email := testhelper.UniqueEmail("repo")
		u := mustSaveUser(t, ctx, email)

		got, err := repo.FindByEmail(ctx, "  "+email+"  ")
		require.NoError(t, err)
		assert.Equal(t, u.ID, got.ID)

		got, err = repo.FindByID(ctx, u.ID)
		require.NoError(t, err)
		assert.Equal(t, email, got.Email.Value())

		_, err = repo.FindByID(ctx, uuid.NewString())
		testhelper.AssertDomainError(t, err, userconstant.CodeUserNotFound)
		_, err = repo.FindByEmail(ctx, "bukan-email")
		testhelper.AssertDomainError(t, err, userconstant.CodeUserNotFound)

		dup, err := userentity.NewUser(userentity.NewUserParams{Email: email, Name: "Dup"}, time.Now())
		require.NoError(t, err)
		testhelper.AssertDomainError(t, repo.Save(ctx, dup), userconstant.CodeUserEmailDuplicate)
	})
}

func TestPostgresAuthSessionRepository(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		u := mustSaveUser(t, ctx, testhelper.UniqueEmail("sess"))
		var solutionID string
		require.NoError(t, persistence.TxFromContext(ctx).QueryRowContext(ctx,
			`INSERT INTO solutions (code, name) VALUES ($1, 'S') RETURNING id`, testhelper.UniqueCode("sess")).Scan(&solutionID))

		repo := persistence.NewPostgresAuthSessionRepository(testDB)
		now := time.Now()
		s, err := sessionentity.NewAuthSession(sessionentity.NewAuthSessionParams{
			UserID: u.ID, SolutionID: solutionID, RefreshTokenHash: "hash-1", IPAddress: "10.1.2.3", TTL: time.Hour,
		}, now)
		require.NoError(t, err)
		require.NoError(t, repo.Create(ctx, s))

		got, err := repo.FindByRefreshTokenHash(ctx, "hash-1")
		require.NoError(t, err)
		assert.Equal(t, s.ID, got.ID)
		require.NotNil(t, got.IPAddress)
		assert.Equal(t, "10.1.2.3", *got.IPAddress)
		assert.Nil(t, got.CompanyID)

		// Rotasi berhasil sekali.
		prev, err := got.Rotate("hash-2", time.Hour, now)
		require.NoError(t, err)
		require.NoError(t, repo.SaveRotation(ctx, got, prev))

		// Pemanggil kedua yang masih memegang hash lama kalah (compare-and-swap).
		stale, err := repo.FindByID(ctx, s.ID)
		require.NoError(t, err)
		stale.RefreshTokenHash = "hash-1"
		prevStale, err := stale.Rotate("hash-3", time.Hour, now)
		require.NoError(t, err)
		testhelper.AssertDomainError(t, repo.SaveRotation(ctx, stale, prevStale), sessionconstant.CodeSessionRotationConflict)

		_, err = repo.FindByRefreshTokenHash(ctx, "hash-1")
		testhelper.AssertDomainError(t, err, sessionconstant.CodeSessionNotFound)

		// Revoke.
		got.Revoke(now)
		require.NoError(t, repo.SaveRevocation(ctx, got))
		revoked, err := repo.FindByID(ctx, s.ID)
		require.NoError(t, err)
		testhelper.AssertDomainError(t, revoked.EnsureUsable(now), sessionconstant.CodeSessionRevoked)
	})
}

func TestPostgresAuthSessionRepository_EmptyIP(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		u := mustSaveUser(t, ctx, testhelper.UniqueEmail("noip"))
		var solutionID string
		require.NoError(t, persistence.TxFromContext(ctx).QueryRowContext(ctx,
			`INSERT INTO solutions (code, name) VALUES ($1, 'S') RETURNING id`, testhelper.UniqueCode("noip")).Scan(&solutionID))

		s, err := sessionentity.NewAuthSession(sessionentity.NewAuthSessionParams{
			UserID: u.ID, SolutionID: solutionID, RefreshTokenHash: "hash-noip", IPAddress: "", TTL: time.Hour,
		}, time.Now())
		require.NoError(t, err)
		require.NoError(t, persistence.NewPostgresAuthSessionRepository(testDB).Create(ctx, s))
	})
}
