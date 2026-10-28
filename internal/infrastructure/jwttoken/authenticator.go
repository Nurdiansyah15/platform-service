package jwttoken

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"platform-service/internal/app/principal"
)

// clockLeeway mentoleransi selisih jam antar server saat cek exp/iat.
const clockLeeway = 30 * time.Second

var errInvalidClaims = errors.New("jwt: claim tidak lengkap / tidak valid")

// Authenticator implements port.TokenAuthenticator untuk token yang diterbitkan Issuer.
//
// Fase 2 hanya memverifikasi kriptografi & claim. Pengecekan sesi/user (Fase 3)
// dan status solution client (Fase 4) ditambahkan di atasnya.
type Authenticator struct {
	keys      *KeyRing
	issuerURL string
	now       func() time.Time
}

func NewAuthenticator(keys *KeyRing, issuerURL string) *Authenticator {
	return &Authenticator{keys: keys, issuerURL: issuerURL, now: time.Now}
}

func (a *Authenticator) Authenticate(ctx context.Context, token string) (*principal.Principal, error) {
	var claims parsedClaims
	_, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("jwt: header kid kosong")
		}
		return a.keys.PublicKey(ctx, kid)
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuer(a.issuerURL),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(clockLeeway),
		jwt.WithTimeFunc(a.now),
	)
	if err != nil {
		return nil, fmt.Errorf("jwt: token tidak valid: %w", err)
	}
	return toPrincipal(&claims)
}

func toPrincipal(c *parsedClaims) (*principal.Principal, error) {
	if c.Subject == "" || len(c.Audience) != 1 {
		return nil, errInvalidClaims
	}
	aud := c.Audience[0]

	switch c.Typ {
	case typUser:
		if c.Sid == "" || aud == "" || aud == audiencePlatform {
			return nil, errInvalidClaims
		}
		return &principal.Principal{
			Kind:         principal.KindUser,
			UserID:       c.Subject,
			SessionID:    c.Sid,
			CompanyID:    c.Cid,
			SolutionCode: aud,
		}, nil

	case typService:
		if aud != audiencePlatform || c.Sol == "" {
			return nil, errInvalidClaims
		}
		scopes := strings.Fields(c.Scope)
		slices.Sort(scopes)
		return &principal.Principal{
			Kind:       principal.KindService,
			ClientID:   c.Subject,
			SolutionID: c.Sol,
			Scopes:     slices.Compact(scopes),
		}, nil

	default:
		return nil, errInvalidClaims
	}
}
