package router

import "platform-service/internal/interfaces/http/handler/api"

// RegisterJWKSRoutes — endpoint publik discovery key (spec §3.16).
func RegisterJWKSRoutes(r Routes, h *api.JWKSHandler) {
	r.V1.GET("/.well-known/jwks.json", h.JWKS)
}
