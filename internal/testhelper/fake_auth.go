package testhelper

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"

	"platform-service/internal/app/principal"
)

// FakeAuthenticator menerbitkan token opaque yang dipetakan langsung ke principal.
// Dipakai handler test sebelum verifier JWT tersedia (Fase 2–4) dan untuk
// skenario yang butuh principal spesifik tanpa alur login.
type FakeAuthenticator struct {
	mu     sync.RWMutex
	tokens map[string]*principal.Principal
}

func NewFakeAuthenticator() *FakeAuthenticator {
	return &FakeAuthenticator{tokens: map[string]*principal.Principal{}}
}

func (f *FakeAuthenticator) Authenticate(_ context.Context, token string) (*principal.Principal, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	p, ok := f.tokens[token]
	if !ok {
		return nil, errors.New("fake authenticator: token tidak dikenal")
	}
	cp := *p
	return &cp, nil
}

// Issue mendaftarkan principal dan mengembalikan token untuknya.
func (f *FakeAuthenticator) Issue(p principal.Principal) string {
	token := "fake-" + uuid.NewString()
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tokens[token] = &p
	return token
}

// UserToken menerbitkan token user biasa (bukan platform admin, tanpa tenant aktif).
func (f *FakeAuthenticator) UserToken() string {
	return f.Issue(principal.Principal{Kind: principal.KindUser, UserID: uuid.NewString(), SessionID: uuid.NewString()})
}

// PlatformAdminToken menerbitkan token user dengan is_platform_admin = TRUE.
func (f *FakeAuthenticator) PlatformAdminToken() string {
	return f.Issue(principal.Principal{Kind: principal.KindUser, UserID: uuid.NewString(), SessionID: uuid.NewString(), IsPlatformAdmin: true})
}

// ServiceToken menerbitkan token service untuk solution tertentu.
func (f *FakeAuthenticator) ServiceToken(solutionID string, scopes ...string) string {
	return f.Issue(principal.Principal{Kind: principal.KindService, ClientID: "client-" + uuid.NewString()[:8], SolutionID: solutionID, Scopes: scopes})
}
