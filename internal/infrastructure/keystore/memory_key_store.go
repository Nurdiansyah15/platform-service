package keystore

import (
	"context"
	"crypto"
	"crypto/rsa"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

const memoryRefPrefix = "mem:"

// MemoryKeyStore menyimpan private key di memori proses. Hanya untuk test —
// key hilang saat proses berhenti.
type MemoryKeyStore struct {
	mu   sync.RWMutex
	keys map[string]*rsa.PrivateKey
}

func NewMemoryKeyStore() *MemoryKeyStore {
	return &MemoryKeyStore{keys: map[string]*rsa.PrivateKey{}}
}

func (s *MemoryKeyStore) Generate(context.Context) (string, string, error) {
	key, pubPEM, err := generateRSAKey()
	if err != nil {
		return "", "", err
	}
	ref := memoryRefPrefix + uuid.NewString()
	s.mu.Lock()
	s.keys[ref] = key
	s.mu.Unlock()
	return ref, pubPEM, nil
}

func (s *MemoryKeyStore) Signer(_ context.Context, ref string) (crypto.Signer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key, ok := s.keys[ref]
	if !ok {
		return nil, fmt.Errorf("keystore: ref %q tidak ditemukan", ref)
	}
	return key, nil
}
