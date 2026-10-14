package solutionclient_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/app/principal"
	"platform-service/internal/app/usecase/solutionclient"
	domainerr "platform-service/internal/domain/errors"
	solconstant "platform-service/internal/domain/solution/constant"
	solentity "platform-service/internal/domain/solution/entity"
	"platform-service/internal/domain/solutionclient/constant"
	"platform-service/internal/domain/solutionclient/entity"
)

type fakeClientRepo struct {
	byClientID map[string]*entity.SolutionClient
}

func (r *fakeClientRepo) Save(context.Context, *entity.SolutionClient) error { return nil }
func (r *fakeClientRepo) FindByID(context.Context, string) (*entity.SolutionClient, error) {
	return nil, domainerr.New(constant.CodeClientNotFound)
}
func (r *fakeClientRepo) FindByClientID(_ context.Context, id string) (*entity.SolutionClient, error) {
	if c, ok := r.byClientID[id]; ok {
		return c, nil
	}
	return nil, domainerr.New(constant.CodeClientNotFound)
}
func (r *fakeClientRepo) TouchLastUsed(context.Context, string, time.Time) error { return nil }

type fakeSolutionRepo struct{ s *solentity.Solution }

func (r *fakeSolutionRepo) Save(context.Context, *solentity.Solution) error { return nil }
func (r *fakeSolutionRepo) FindByID(_ context.Context, id string) (*solentity.Solution, error) {
	if r.s != nil && r.s.ID == id {
		return r.s, nil
	}
	return nil, domainerr.New(solconstant.CodeSolutionNotFound)
}
func (r *fakeSolutionRepo) FindByCode(context.Context, string) (*solentity.Solution, error) {
	return nil, domainerr.New(solconstant.CodeSolutionNotFound)
}

func setup(t *testing.T, clientScopes ...string) (*solutionclient.SolutionClientUseCase, *entity.SolutionClient, *solentity.Solution) {
	t.Helper()
	sol, err := solentity.NewSolution(solentity.NewSolutionParams{Code: "easy-hr", Name: "Easy HR"})
	require.NoError(t, err)
	c, err := entity.NewSolutionClient(entity.NewSolutionClientParams{
		SolutionID: sol.ID, Name: "Backend", ClientID: "cli_1", ClientSecretHash: "h", Scopes: clientScopes,
	}, time.Now())
	require.NoError(t, err)

	uc := solutionclient.NewSolutionClientUseCase(solutionclient.SolutionClientDeps{
		Repo:      &fakeClientRepo{byClientID: map[string]*entity.SolutionClient{c.ClientID: c}},
		Solutions: &fakeSolutionRepo{s: sol},
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	return uc, c, sol
}

func servicePrincipal(clientID, solutionID string, scopes ...string) *principal.Principal {
	return &principal.Principal{Kind: principal.KindService, ClientID: clientID, SolutionID: solutionID, Scopes: scopes}
}

func TestResolveServicePrincipal_IntersectsWithCurrentScopes(t *testing.T) {
	uc, c, sol := setup(t, "usage:write")

	// Token diterbitkan saat client masih punya catalog:sync; scope itu sudah dicabut.
	p, err := uc.ResolveServicePrincipal(context.Background(), servicePrincipal(c.ClientID, sol.ID, "catalog:sync", "usage:write"))
	require.NoError(t, err)
	assert.Equal(t, []string{"usage:write"}, p.Scopes)
}

func TestResolveServicePrincipal_Rejects(t *testing.T) {
	ctx := context.Background()

	uc, c, sol := setup(t, "usage:write")
	_, err := uc.ResolveServicePrincipal(ctx, servicePrincipal("cli_lain", sol.ID))
	assert.Error(t, err, "client tidak dikenal")

	_, err = uc.ResolveServicePrincipal(ctx, servicePrincipal(c.ClientID, "solution-lain"))
	assert.Error(t, err, "claim sol tidak cocok")

	c.Revoke(time.Now())
	_, err = uc.ResolveServicePrincipal(ctx, servicePrincipal(c.ClientID, sol.ID))
	assert.Error(t, err, "client revoked")

	uc, c, sol = setup(t)
	sol.Status = "inactive"
	_, err = uc.ResolveServicePrincipal(ctx, servicePrincipal(c.ClientID, sol.ID))
	assert.Error(t, err, "solution inactive")
}
