package jwttoken

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"platform-service/internal/app/port"
)

// Issuer implements port.TokenIssuer.
type Issuer struct {
	keys      *KeyRing
	issuerURL string
	ttl       time.Duration
	now       func() time.Time
}

func NewIssuer(keys *KeyRing, issuerURL string, ttl time.Duration) *Issuer {
	return &Issuer{keys: keys, issuerURL: issuerURL, ttl: ttl, now: time.Now}
}

func (i *Issuer) IssueUserToken(ctx context.Context, c port.UserTokenClaims) (port.IssuedToken, error) {
	return i.sign(ctx, payload{
		Sub: c.UserID,
		Aud: c.SolutionCode,
		Typ: typUser,
		Cid: c.CompanyID,
		Sid: c.SessionID,
	})
}

func (i *Issuer) IssueServiceToken(ctx context.Context, c port.ServiceTokenClaims) (port.IssuedToken, error) {
	return i.sign(ctx, payload{
		Sub:   c.ClientID,
		Aud:   audiencePlatform,
		Typ:   typService,
		Sol:   c.SolutionID,
		Scope: strings.Join(c.Scopes, " "),
	})
}

// sign membentuk JWT RS256 secara manual agar bisa memakai crypto.Signer
// (private key bisa berada di KMS, tidak harus *rsa.PrivateKey di memori).
func (i *Issuer) sign(ctx context.Context, p payload) (port.IssuedToken, error) {
	kid, signer, err := i.keys.ActiveSigner(ctx)
	if err != nil {
		return port.IssuedToken{}, err
	}

	now := i.now().UTC()
	exp := now.Add(i.ttl)
	p.Iss = i.issuerURL
	p.Iat = now.Unix()
	p.Exp = exp.Unix()
	p.Jti = uuid.NewString()

	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid})
	if err != nil {
		return port.IssuedToken{}, fmt.Errorf("jwt: marshal header: %w", err)
	}
	body, err := json.Marshal(p)
	if err != nil {
		return port.IssuedToken{}, fmt.Errorf("jwt: marshal claims: %w", err)
	}

	enc := base64.RawURLEncoding
	signingInput := enc.EncodeToString(header) + "." + enc.EncodeToString(body)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := signer.Sign(rand.Reader, digest[:], crypto.SHA256)
	if err != nil {
		return port.IssuedToken{}, fmt.Errorf("jwt: sign: %w", err)
	}

	return port.IssuedToken{Token: signingInput + "." + enc.EncodeToString(sig), ExpiresAt: exp}, nil
}
