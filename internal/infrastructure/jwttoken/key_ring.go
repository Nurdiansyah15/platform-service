// Package jwttoken menandatangani dan memverifikasi access token JWT (RS256)
// memakai signing_keys + port.KeyStore.
package jwttoken

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"sync"
	"time"

	"platform-service/internal/app/port"
	signingrepo "platform-service/internal/domain/signingkey/repository"
)

// keyCacheTTL membatasi berapa lama perubahan status key (rotasi / retire)
// butuh waktu untuk berlaku di instance yang sedang berjalan.
const keyCacheTTL = time.Minute

var errKeyNotPublished = errors.New("jwt: signing key tidak aktif / sudah retired")

// KeyRing menyediakan signer key active dan public key per kid, dengan cache.
type KeyRing struct {
	repo  signingrepo.SigningKeyRepository
	store port.KeyStore
	now   func() time.Time

	mu       sync.Mutex
	active   *cachedSigner
	verifier map[string]cachedPublicKey
}

type cachedSigner struct {
	kid      string
	signer   crypto.Signer
	loadedAt time.Time
}

type cachedPublicKey struct {
	key      *rsa.PublicKey // nil = kid tidak dikenal / tidak dipublikasikan
	loadedAt time.Time
}

func NewKeyRing(repo signingrepo.SigningKeyRepository, store port.KeyStore) *KeyRing {
	return &KeyRing{repo: repo, store: store, now: time.Now, verifier: map[string]cachedPublicKey{}}
}

// ActiveSigner mengembalikan kid dan signer dari key active.
func (r *KeyRing) ActiveSigner(ctx context.Context) (string, crypto.Signer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.active != nil && r.now().Sub(r.active.loadedAt) < keyCacheTTL {
		return r.active.kid, r.active.signer, nil
	}
	k, err := r.repo.FindActive(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("jwt: load active key: %w", err)
	}
	signer, err := r.store.Signer(ctx, k.PrivateKeyRef)
	if err != nil {
		return "", nil, fmt.Errorf("jwt: load signer %s: %w", k.KID, err)
	}
	r.active = &cachedSigner{kid: k.KID, signer: signer, loadedAt: r.now()}
	return k.KID, signer, nil
}

// PublicKey mengembalikan public key untuk kid yang masih dipublikasikan (active / retiring).
func (r *KeyRing) PublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if c, ok := r.verifier[kid]; ok && r.now().Sub(c.loadedAt) < keyCacheTTL {
		if c.key == nil {
			return nil, errKeyNotPublished
		}
		return c.key, nil
	}

	k, err := r.repo.FindByKID(ctx, kid)
	if err != nil {
		// Kid tak dikenal tidak di-cache agar spam kid acak tidak menumpuk di memori.
		return nil, fmt.Errorf("jwt: load key %s: %w", kid, err)
	}
	var pub *rsa.PublicKey
	if k.Status.IsPublished() {
		if pub, err = ParseRSAPublicKeyPEM(k.PublicKeyPEM); err != nil {
			return nil, err
		}
	}
	r.verifier[kid] = cachedPublicKey{key: pub, loadedAt: r.now()}
	if pub == nil {
		return nil, errKeyNotPublished
	}
	return pub, nil
}

// ParseRSAPublicKeyPEM mem-parse public key PKIX PEM.
func ParseRSAPublicKeyPEM(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("jwt: public key bukan PEM")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("jwt: parse public key: %w", err)
	}
	pub, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("jwt: public key bukan RSA")
	}
	return pub, nil
}
