package middleware

import (
	"slices"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS mengizinkan origin dari config (CORS_ALLOW_ORIGINS). "*" berarti semua origin;
// credentials hanya diaktifkan bila origin dibatasi.
func CORS(allowOrigins []string) gin.HandlerFunc {
	allowAll := len(allowOrigins) == 0 || slices.Contains(allowOrigins, "*")
	cfg := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-Id", "Accept-Language"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-Id", "X-Client-Request-Id"},
		AllowCredentials: !allowAll,
		MaxAge:           12 * time.Hour,
	}
	if allowAll {
		cfg.AllowAllOrigins = true
	} else {
		cfg.AllowOrigins = allowOrigins
	}
	return cors.New(cfg)
}
