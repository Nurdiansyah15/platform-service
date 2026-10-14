package keystore_test

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/app/port"
	"platform-service/internal/infrastructure/keystore"
)

// assertSignerMatchesPublicKey memastikan signature dari Signer terverifikasi oleh public key PEM.
func assertSignerMatchesPublicKey(t *testing.T, ks port.KeyStore, ref, pubPEM string) {
	t.Helper()
	signer, err := ks.Signer(context.Background(), ref)
	require.NoError(t, err)

	digest := sha256.Sum256([]byte("payload"))
	sig, err := signer.Sign(nil, digest[:], crypto.SHA256)
	require.NoError(t, err)

	block, _ := pem.Decode([]byte(pubPEM))
	require.NotNil(t, block)
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	require.NoError(t, err)
	assert.NoError(t, rsa.VerifyPKCS1v15(pub.(*rsa.PublicKey), crypto.SHA256, digest[:], sig))
}

func TestMemoryKeyStore(t *testing.T) {
	ks := keystore.NewMemoryKeyStore()
	ref, pubPEM, err := ks.Generate(context.Background())
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(ref, "mem:"))
	assertSignerMatchesPublicKey(t, ks, ref, pubPEM)

	_, err = ks.Signer(context.Background(), "mem:tidak-ada")
	assert.Error(t, err)
}

func TestFileKeyStore(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "keys")
	ks, err := keystore.NewFileKeyStore(dir)
	require.NoError(t, err)

	ref, pubPEM, err := ks.Generate(context.Background())
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(ref, "file:"))
	assertSignerMatchesPublicKey(t, ks, ref, pubPEM)

	info, err := os.Stat(filepath.Join(dir, strings.TrimPrefix(ref, "file:")))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "private key harus 0600")

	// Store baru di direktori yang sama tetap bisa memakai ref lama (restart aplikasi).
	reopened, err := keystore.NewFileKeyStore(dir)
	require.NoError(t, err)
	assertSignerMatchesPublicKey(t, reopened, ref, pubPEM)
}

func TestFileKeyStore_RejectsPathTraversal(t *testing.T) {
	ks, err := keystore.NewFileKeyStore(t.TempDir())
	require.NoError(t, err)

	for _, ref := range []string{"file:../../etc/passwd", "file:/etc/passwd", "mem:abc", "file:"} {
		_, err := ks.Signer(context.Background(), ref)
		assert.Error(t, err, ref)
	}
}
