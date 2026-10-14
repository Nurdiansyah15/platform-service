// Package keystore berisi implementasi port.KeyStore.
//
//   - FileKeyStore:   private key PEM di direktori lokal (dev / on_prem sederhana)
//   - MemoryKeyStore: in-memory, untuk test
//
// Implementasi produksi cloud (KMS / Vault) mengikuti kontrak yang sama.
package keystore

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

const rsaKeyBits = 2048

func generateRSAKey() (*rsa.PrivateKey, string, error) {
	key, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return nil, "", fmt.Errorf("keystore: generate rsa key: %w", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, "", fmt.Errorf("keystore: marshal public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	return key, string(pubPEM), nil
}
