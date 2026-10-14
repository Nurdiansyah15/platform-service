package api_test

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/app/port"
	"platform-service/internal/testhelper"
)

const jwksPath = "/api/v1/.well-known/jwks.json"

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func getJWKS(t *testing.T) map[string]jwk {
	t.Helper()
	w := testSrv.GET(jwksPath)
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp struct {
		Keys []jwk `json:"keys"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	out := map[string]jwk{}
	for _, k := range resp.Keys {
		out[k.Kid] = k
	}
	return out
}

func jwkToRSA(t *testing.T, k jwk) *rsa.PublicKey {
	t.Helper()
	n, err := base64.RawURLEncoding.DecodeString(k.N)
	require.NoError(t, err)
	e, err := base64.RawURLEncoding.DecodeString(k.E)
	require.NoError(t, err)
	return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
}

func activeKID(t *testing.T) string {
	t.Helper()
	var kid string
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT kid FROM signing_keys WHERE status = 'active'`).Scan(&kid))
	return kid
}

// ── JWKS ─────────────────────────────────────────────────────────────────────

func TestJWKS_Get_PublicWithoutToken(t *testing.T) {
	w := testSrv.GET(jwksPath)
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, "public, max-age=300", w.Header().Get("Cache-Control"))

	// Format RFC 7517 — tanpa envelope respond.
	var raw map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(w.Body).Decode(&raw))
	assert.Contains(t, raw, "keys")
	assert.NotContains(t, raw, "status")
}

func TestJWKS_Get_ContainsActiveKeyCreatedAtStartup(t *testing.T) {
	keys := getJWKS(t)
	k, ok := keys[activeKID(t)]
	require.True(t, ok, "key active harus ada di JWKS")
	assert.Equal(t, "RSA", k.Kty)
	assert.Equal(t, "sig", k.Use)
	assert.Equal(t, "RS256", k.Alg)
	assert.Equal(t, "AQAB", k.E)
	assert.NotEmpty(t, k.N)
}

func TestJWKS_Get_OnlyActiveAndRetiring(t *testing.T) {
	retiring := testhelper.MustSeedSigningKey(t, testSrv, "retiring")
	retired := testhelper.MustSeedSigningKey(t, testSrv, "retired")

	keys := getJWKS(t)
	assert.Contains(t, keys, retiring)
	assert.NotContains(t, keys, retired)
}

func TestJWKS_Get_VerifiesIssuedToken(t *testing.T) {
	issued, err := testSrv.Issuer.IssueUserToken(context.Background(), port.UserTokenClaims{
		UserID: "u-1", SolutionCode: "easy-hr", SessionID: "s-1",
	})
	require.NoError(t, err)

	keys := getJWKS(t)
	// Simulasi solution: verifikasi token hanya dengan JWKS publik.
	parsed, err := jwt.Parse(issued.Token, func(tok *jwt.Token) (any, error) {
		k, ok := keys[tok.Header["kid"].(string)]
		require.True(t, ok)
		return jwkToRSA(t, k), nil
	}, jwt.WithValidMethods([]string{"RS256"}))
	require.NoError(t, err)
	assert.True(t, parsed.Valid)
}

// ── Startup ──────────────────────────────────────────────────────────────────

func TestSigningKey_StartupKeepsSingleActiveKey(t *testing.T) {
	// testSrv dan testOnPremSrv sama-sama menjalankan EnsureActiveKey di DB yang sama.
	var n int
	require.NoError(t, testSrv.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM signing_keys WHERE status = 'active'`).Scan(&n))
	assert.Equal(t, 1, n)
}

// ── JWT pada endpoint terproteksi ────────────────────────────────────────────

func TestAuth_RealJWTAcceptedOnProtectedRoute(t *testing.T) {
	issued, err := testSrv.Issuer.IssueServiceToken(context.Background(), port.ServiceTokenClaims{
		ClientID: "client-1", SolutionID: "00000000-0000-0000-0000-000000000001", Scopes: []string{"catalog:sync"},
	})
	require.NoError(t, err)

	w := testSrv.GET("/api/v1/solutions", testhelper.BearerHeader(issued.Token))
	assert.Equal(t, 200, w.Code, w.Body.String())
}

func TestAuth_InvalidJWTRejected(t *testing.T) {
	w := testSrv.GET("/api/v1/solutions", testhelper.BearerHeader("eyJhbGciOiJSUzI1NiJ9.e30.c2ln"))
	require.Equal(t, 401, w.Code, w.Body.String())

	var msg string
	require.NoError(t, json.Unmarshal(testhelper.DecodeError(t, w).Errors, &msg))
	assert.Equal(t, "Token is invalid or expired.", msg)
}
