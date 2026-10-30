package port

import "context"

// OriginAllowlist menentukan origin browser yang boleh memanggil API (CORS) di luar
// CORS_ALLOW_ORIGINS, yaitu origin URL aplikasi & redirect URI solution active.
type OriginAllowlist interface {
	IsAllowedOrigin(ctx context.Context, origin string) bool
}
