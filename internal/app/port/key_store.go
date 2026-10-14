package port

import (
	"context"
	"crypto"
)

// KeyStore mengelola private key penandatangan JWT dengan gaya KMS: private key
// dibuat dan disimpan di dalam store, aplikasi hanya memegang referensinya
// (signing_keys.private_key_ref) dan public key.
type KeyStore interface {
	// Generate membuat key pair RSA baru dan mengembalikan referensi private key
	// serta public key dalam format PEM (PKIX).
	Generate(ctx context.Context) (privateKeyRef string, publicKeyPEM string, err error)
	// Signer mengembalikan penandatangan untuk referensi tersebut.
	Signer(ctx context.Context, privateKeyRef string) (crypto.Signer, error)
}
