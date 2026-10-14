package port

import (
	"context"

	"platform-service/internal/app/principal"
)

// TokenAuthenticator memverifikasi bearer token (user atau service) dan
// mengembalikan principal-nya. Error apa pun dianggap token tidak valid (401).
// Implementasi JWT menyusul di Fase 2–4.
type TokenAuthenticator interface {
	Authenticate(ctx context.Context, token string) (*principal.Principal, error)
}
