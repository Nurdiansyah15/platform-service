package jwttoken_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/signingkey/constant"
	"platform-service/internal/domain/signingkey/entity"
	"platform-service/internal/infrastructure/jwttoken"
	"platform-service/internal/infrastructure/keystore"
)

const issuerURL = "https://platform.test"

// fakeKeyRepo adalah SigningKeyRepository in-memory.
type fakeKeyRepo struct{ keys []*entity.SigningKey }

func (r *fakeKeyRepo) Save(_ context.Context, k *entity.SigningKey) error {
	r.keys = append(r.keys, k)
	return nil
}

func (r *fakeKeyRepo) FindActive(context.Context) (*entity.SigningKey, error) {
	for _, k := range r.keys {
		if k.IsActive() {
			return k, nil
		}
	}
	return nil, domainerr.New(constant.CodeSigningKeyNotFound)
}

func (r *fakeKeyRepo) FindByKID(_ context.Context, kid string) (*entity.SigningKey, error) {
	for _, k := range r.keys {
		if k.KID == kid {
			return k, nil
		}
	}
	return nil, domainerr.New(constant.CodeSigningKeyNotFound)
}

func (r *fakeKeyRepo) ListPublished(context.Context) ([]*entity.SigningKey, error) {
	return r.keys, nil
}

type fixture struct {
	repo   *fakeKeyRepo
	store  *keystore.MemoryKeyStore
	issuer *jwttoken.Issuer
	authn  *jwttoken.Authenticator
}

func newFixture(t *testing.T, ttl time.Duration) *fixture {
	t.Helper()
	f := &fixture{repo: &fakeKeyRepo{}, store: keystore.NewMemoryKeyStore()}
	f.addActiveKey(t)
	ring := jwttoken.NewKeyRing(f.repo, f.store)
	f.issuer = jwttoken.NewIssuer(ring, issuerURL, ttl)
	f.authn = jwttoken.NewAuthenticator(ring, issuerURL)
	return f
}

func (f *fixture) addActiveKey(t *testing.T) *entity.SigningKey {
	t.Helper()
	ref, pub, err := f.store.Generate(context.Background())
	require.NoError(t, err)
	k, err := entity.NewSigningKey(pub, ref, time.Now())
	require.NoError(t, err)
	require.NoError(t, f.repo.Save(context.Background(), k))
	return k
}

func decodeSegment(t *testing.T, token string, idx int) map[string]any {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(strings.Split(token, ".")[idx])
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

// ── Round trip ───────────────────────────────────────────────────────────────

func TestUserToken_RoundTrip(t *testing.T) {
	f := newFixture(t, 15*time.Minute)
	issued, err := f.issuer.IssueUserToken(context.Background(), port.UserTokenClaims{
		UserID: "u-1", SolutionCode: "easy-hr", TenantID: "t-1", SessionID: "s-1",
	})
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), issued.ExpiresAt, 5*time.Second)

	header := decodeSegment(t, issued.Token, 0)
	assert.Equal(t, "RS256", header["alg"])
	assert.Equal(t, f.repo.keys[0].KID, header["kid"])

	claims := decodeSegment(t, issued.Token, 1)
	assert.Equal(t, issuerURL, claims["iss"])
	assert.Equal(t, "easy-hr", claims["aud"], "aud ditulis sebagai string tunggal")
	assert.Equal(t, "user", claims["typ"])
	assert.NotEmpty(t, claims["jti"])
	assert.NotContains(t, claims, "scope")

	p, err := f.authn.Authenticate(context.Background(), issued.Token)
	require.NoError(t, err)
	assert.Equal(t, &principal.Principal{
		Kind: principal.KindUser, UserID: "u-1", SessionID: "s-1", TenantID: "t-1", SolutionCode: "easy-hr",
	}, p)
}

func TestUserToken_WithoutTenant(t *testing.T) {
	f := newFixture(t, time.Minute)
	issued, err := f.issuer.IssueUserToken(context.Background(), port.UserTokenClaims{UserID: "u-1", SolutionCode: "easy-hr", SessionID: "s-1"})
	require.NoError(t, err)
	assert.NotContains(t, decodeSegment(t, issued.Token, 1), "tid")

	p, err := f.authn.Authenticate(context.Background(), issued.Token)
	require.NoError(t, err)
	assert.False(t, p.HasActiveTenant())
}

func TestServiceToken_RoundTrip(t *testing.T) {
	f := newFixture(t, time.Minute)
	issued, err := f.issuer.IssueServiceToken(context.Background(), port.ServiceTokenClaims{
		ClientID: "c-1", SolutionID: "sol-1", Scopes: []string{"usage:write", "authz:check", "usage:write"},
	})
	require.NoError(t, err)

	claims := decodeSegment(t, issued.Token, 1)
	assert.Equal(t, "platform", claims["aud"])
	assert.Equal(t, "service", claims["typ"])
	assert.Equal(t, "usage:write authz:check usage:write", claims["scope"])

	p, err := f.authn.Authenticate(context.Background(), issued.Token)
	require.NoError(t, err)
	assert.Equal(t, principal.KindService, p.Kind)
	assert.Equal(t, "c-1", p.ClientID)
	assert.Equal(t, "sol-1", p.SolutionID)
	assert.Equal(t, []string{"authz:check", "usage:write"}, p.Scopes)
}

// ── Rotasi ───────────────────────────────────────────────────────────────────

func TestRetiringKeyStillVerifies_RetiredKeyRejected(t *testing.T) {
	f := newFixture(t, time.Minute)
	old := f.repo.keys[0]
	issued, err := f.issuer.IssueUserToken(context.Background(), port.UserTokenClaims{UserID: "u", SolutionCode: "x", SessionID: "s"})
	require.NoError(t, err)

	require.NoError(t, old.StartRetiring())
	f.addActiveKey(t)
	// Authenticator baru = cache kosong (setara setelah keyCacheTTL lewat).
	authn := jwttoken.NewAuthenticator(jwttoken.NewKeyRing(f.repo, f.store), issuerURL)
	_, err = authn.Authenticate(context.Background(), issued.Token)
	assert.NoError(t, err, "token dari key retiring masih valid")

	require.NoError(t, old.Retire(time.Now()))
	authn = jwttoken.NewAuthenticator(jwttoken.NewKeyRing(f.repo, f.store), issuerURL)
	_, err = authn.Authenticate(context.Background(), issued.Token)
	assert.Error(t, err, "token dari key retired ditolak")
}

// ── Token ditolak ────────────────────────────────────────────────────────────

func TestAuthenticate_Rejects(t *testing.T) {
	f := newFixture(t, time.Minute)
	ctx := context.Background()
	valid, err := f.issuer.IssueUserToken(ctx, port.UserTokenClaims{UserID: "u", SolutionCode: "x", SessionID: "s"})
	require.NoError(t, err)

	parts := strings.Split(valid.Token, ".")
	tamperedClaims := decodeSegment(t, valid.Token, 1)
	tamperedClaims["sub"] = "admin"
	tamperedBody, _ := json.Marshal(tamperedClaims)
	tampered := parts[0] + "." + base64.RawURLEncoding.EncodeToString(tamperedBody) + "." + parts[2]

	hs256, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": issuerURL, "sub": "u", "aud": "x", "sid": "s", "typ": "user",
		"exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix(),
	}).SignedString([]byte("secret"))
	require.NoError(t, err)

	otherIssuer := jwttoken.NewIssuer(jwttoken.NewKeyRing(f.repo, f.store), "https://evil.test", time.Minute)
	wrongIss, err := otherIssuer.IssueUserToken(ctx, port.UserTokenClaims{UserID: "u", SolutionCode: "x", SessionID: "s"})
	require.NoError(t, err)

	expiredIssuer := jwttoken.NewIssuer(jwttoken.NewKeyRing(f.repo, f.store), issuerURL, -time.Hour)
	expired, err := expiredIssuer.IssueUserToken(ctx, port.UserTokenClaims{UserID: "u", SolutionCode: "x", SessionID: "s"})
	require.NoError(t, err)

	noSession, err := f.issuer.IssueUserToken(ctx, port.UserTokenClaims{UserID: "u", SolutionCode: "x"})
	require.NoError(t, err)

	userWithPlatformAud, err := f.issuer.IssueUserToken(ctx, port.UserTokenClaims{UserID: "u", SolutionCode: "platform", SessionID: "s"})
	require.NoError(t, err)

	serviceWithoutSolution, err := f.issuer.IssueServiceToken(ctx, port.ServiceTokenClaims{ClientID: "c"})
	require.NoError(t, err)

	unknownKid := strings.Replace(valid.Token, parts[0], base64.RawURLEncoding.EncodeToString(
		[]byte(`{"alg":"RS256","typ":"JWT","kid":"tidak-ada"}`)), 1)

	tests := map[string]string{
		"sampah":                   "bukan.jwt",
		"payload diubah":           tampered,
		"alg HS256":                hs256,
		"issuer lain":              wrongIss.Token,
		"kedaluwarsa":              expired.Token,
		"user tanpa sid":           noSession.Token,
		"user dengan aud platform": userWithPlatformAud.Token,
		"service tanpa sol":        serviceWithoutSolution.Token,
		"kid tidak dikenal":        unknownKid,
	}
	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := f.authn.Authenticate(ctx, token)
			assert.Error(t, err)
		})
	}
}

func TestIssue_WithoutActiveKey(t *testing.T) {
	repo := &fakeKeyRepo{}
	issuer := jwttoken.NewIssuer(jwttoken.NewKeyRing(repo, keystore.NewMemoryKeyStore()), issuerURL, time.Minute)
	_, err := issuer.IssueUserToken(context.Background(), port.UserTokenClaims{UserID: "u", SolutionCode: "x", SessionID: "s"})
	assert.Error(t, err)
}
