package solution_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/app/usecase/solution"
	"platform-service/internal/domain/solution/entity"
	solutionrepo "platform-service/internal/domain/solution/repository"
)

// fakeActiveRepo hanya mengimplementasikan ListActive; method lain tidak dipakai allowlist.
type fakeActiveRepo struct {
	solutionrepo.SolutionRepository
	solutions []*entity.Solution
	err       error
	calls     int
}

func (r *fakeActiveRepo) ListActive(context.Context) ([]*entity.Solution, error) {
	r.calls++
	return r.solutions, r.err
}

func solutionWithLaunch(t *testing.T, launch string) *entity.Solution {
	t.Helper()
	s, err := entity.NewSolution(entity.NewSolutionParams{Code: "kasir", Name: "Kasir", Links: entity.AppLinks{LaunchURL: &launch}})
	require.NoError(t, err)
	return s
}

func newAllowlist(repo *fakeActiveRepo, ttl time.Duration) *solution.OriginAllowlist {
	return solution.NewOriginAllowlist(repo, ttl, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestOriginAllowlist_MatchesNormalizedOrigin(t *testing.T) {
	repo := &fakeActiveRepo{solutions: []*entity.Solution{solutionWithLaunch(t, "https://kasir.example.com/app")}}
	a := newAllowlist(repo, time.Minute)
	ctx := context.Background()

	assert.True(t, a.IsAllowedOrigin(ctx, "https://kasir.example.com"))
	assert.True(t, a.IsAllowedOrigin(ctx, "https://KASIR.example.com:443"))
	assert.False(t, a.IsAllowedOrigin(ctx, "http://kasir.example.com"))
	assert.False(t, a.IsAllowedOrigin(ctx, "https://other.example.com"))
	assert.False(t, a.IsAllowedOrigin(ctx, "null"))
	assert.Equal(t, 1, repo.calls, "dimuat sekali selama TTL")
}

func TestOriginAllowlist_InvalidateReloads(t *testing.T) {
	repo := &fakeActiveRepo{}
	a := newAllowlist(repo, time.Hour)
	ctx := context.Background()
	require.False(t, a.IsAllowedOrigin(ctx, "https://kasir.example.com"))

	repo.solutions = []*entity.Solution{solutionWithLaunch(t, "https://kasir.example.com/")}
	assert.False(t, a.IsAllowedOrigin(ctx, "https://kasir.example.com"), "masih cache lama")

	a.Invalidate()
	assert.True(t, a.IsAllowedOrigin(ctx, "https://kasir.example.com"))
}

func TestOriginAllowlist_ExpiresAfterTTL(t *testing.T) {
	repo := &fakeActiveRepo{}
	a := newAllowlist(repo, time.Millisecond)
	ctx := context.Background()
	require.False(t, a.IsAllowedOrigin(ctx, "https://kasir.example.com"))

	repo.solutions = []*entity.Solution{solutionWithLaunch(t, "https://kasir.example.com/")}
	time.Sleep(5 * time.Millisecond)
	assert.True(t, a.IsAllowedOrigin(ctx, "https://kasir.example.com"))
}

func TestOriginAllowlist_QueryErrorKeepsLastList(t *testing.T) {
	repo := &fakeActiveRepo{solutions: []*entity.Solution{solutionWithLaunch(t, "https://kasir.example.com/")}}
	a := newAllowlist(repo, time.Hour)
	ctx := context.Background()
	require.True(t, a.IsAllowedOrigin(ctx, "https://kasir.example.com"))

	repo.err = errors.New("db down")
	a.Invalidate()
	assert.True(t, a.IsAllowedOrigin(ctx, "https://kasir.example.com"))
}

func TestOriginAllowlist_QueryErrorBeforeFirstLoadDenies(t *testing.T) {
	a := newAllowlist(&fakeActiveRepo{err: errors.New("db down")}, time.Hour)
	assert.False(t, a.IsAllowedOrigin(context.Background(), "https://kasir.example.com"))
}
