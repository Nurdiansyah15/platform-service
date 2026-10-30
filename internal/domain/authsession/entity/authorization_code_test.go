package entity_test

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/authsession/constant"
	"platform-service/internal/domain/authsession/entity"
	"platform-service/internal/testhelper"
)

func s256(v string) string {
	sum := sha256.Sum256([]byte(v))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

const testVerifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

func newCode(t *testing.T, now time.Time) *entity.AuthorizationCode {
	t.Helper()
	c, err := entity.NewAuthorizationCode(entity.NewAuthorizationCodeParams{
		CodeHash: "h", UserID: "u", SolutionID: "s", ParentSessionID: "p",
		RedirectURI: "https://app.example.com/cb", CodeChallenge: s256(testVerifier),
		CodeChallengeMethod: constant.CodeChallengeMethodS256, TTL: time.Minute,
	}, now)
	require.NoError(t, err)
	return c
}

func TestNewAuthorizationCode_RejectsBadChallenge(t *testing.T) {
	for _, tc := range []struct{ name, challenge, method string }{
		{"plain method", s256(testVerifier), "plain"},
		{"too short", "abc", constant.CodeChallengeMethodS256},
		{"not base64url", strings.Repeat("!", 43), constant.CodeChallengeMethodS256},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := entity.NewAuthorizationCode(entity.NewAuthorizationCodeParams{
				CodeHash: "h", UserID: "u", SolutionID: "s", ParentSessionID: "p",
				RedirectURI: "https://app.example.com/cb", CodeChallenge: tc.challenge,
				CodeChallengeMethod: tc.method, TTL: time.Minute,
			}, time.Now())
			testhelper.AssertDomainError(t, err, constant.CodeAuthCodeChallengeInvalid)
		})
	}
}

func TestAuthorizationCode_Redeem(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name     string
		verifier string
		redirect string
		at       time.Time
		used     bool
		ok       bool
	}{
		{"valid", testVerifier, "https://app.example.com/cb", now, false, true},
		{"wrong verifier", strings.Repeat("a", 43), "https://app.example.com/cb", now, false, false},
		{"short verifier", "abc", "https://app.example.com/cb", now, false, false},
		{"redirect mismatch", testVerifier, "https://app.example.com/cb/", now, false, false},
		{"expired", testVerifier, "https://app.example.com/cb", now.Add(time.Minute), false, false},
		{"already used", testVerifier, "https://app.example.com/cb", now, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newCode(t, now)
			if tc.used {
				used := now
				c.UsedAt = &used
			}
			err := c.Redeem(tc.verifier, tc.redirect, tc.at)
			if tc.ok {
				require.NoError(t, err)
				assert.NotNil(t, c.UsedAt)
				return
			}
			testhelper.AssertDomainError(t, err, constant.CodeInvalidGrant)
		})
	}
}

// RFC 7636 Appendix B.
func TestVerifyPKCE_RFCVector(t *testing.T) {
	assert.True(t, entity.VerifyPKCE("E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", testVerifier))
}
