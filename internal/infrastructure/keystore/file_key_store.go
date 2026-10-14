package keystore

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

const fileRefPrefix = "file:"

var fileRefName = regexp.MustCompile(`^[a-f0-9-]{36}\.pem$`)

// FileKeyStore menyimpan private key (PKCS#8 PEM, mode 0600) di satu direktori.
// private_key_ref berbentuk "file:<uuid>.pem" — relatif terhadap direktori, sehingga
// direktori bisa dipindah tanpa mengubah DB.
type FileKeyStore struct {
	dir string
}

func NewFileKeyStore(dir string) (*FileKeyStore, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("keystore: direktori wajib diisi")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("keystore: buat direktori %s: %w", dir, err)
	}
	return &FileKeyStore{dir: dir}, nil
}

func (s *FileKeyStore) Generate(context.Context) (string, string, error) {
	key, pubPEM, err := generateRSAKey()
	if err != nil {
		return "", "", err
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", "", fmt.Errorf("keystore: marshal private key: %w", err)
	}
	name := uuid.NewString() + ".pem"
	data := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	// O_EXCL: jangan pernah menimpa key yang sudah ada.
	f, err := os.OpenFile(filepath.Join(s.dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", "", fmt.Errorf("keystore: tulis private key: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return "", "", fmt.Errorf("keystore: tulis private key: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", "", fmt.Errorf("keystore: tutup file private key: %w", err)
	}
	return fileRefPrefix + name, pubPEM, nil
}

func (s *FileKeyStore) Signer(_ context.Context, ref string) (crypto.Signer, error) {
	name, ok := strings.CutPrefix(ref, fileRefPrefix)
	if !ok || !fileRefName.MatchString(name) {
		return nil, fmt.Errorf("keystore: ref %q bukan ref file yang valid", ref)
	}
	data, err := os.ReadFile(filepath.Join(s.dir, name))
	if err != nil {
		return nil, fmt.Errorf("keystore: baca private key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("keystore: file %s bukan PEM", name)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("keystore: parse private key: %w", err)
	}
	signer, ok := parsed.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("keystore: key %s tidak bisa menandatangani", name)
	}
	return signer, nil
}
