package middleware

import (
	"slices"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"platform-service/internal/app/port"
)

// CORS mengizinkan origin dari config (CORS_ALLOW_ORIGINS) ditambah origin solution yang
// terdaftar di console (URL aplikasi & redirect URI, lewat dynamic). "*" berarti semua origin;
// credentials hanya diaktifkan bila origin dibatasi. Origin lain ditolak 403.
func CORS(allowOrigins []string, dynamic port.OriginAllowlist) gin.HandlerFunc {
	allowAll := slices.Contains(allowOrigins, "*") || (len(allowOrigins) == 0 && dynamic == nil)
	cfg := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-Id", "Accept-Language"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-Id", "X-Client-Request-Id"},
		AllowCredentials: !allowAll,
		MaxAge:           12 * time.Hour,
	}
	if allowAll {
		cfg.AllowAllOrigins = true
		return cors.New(cfg)
	}
	cfg.AllowOrigins = allowOrigins
	if dynamic != nil {
		cfg.AllowOriginWithContextFunc = func(c *gin.Context, origin string) bool {
			return dynamic.IsAllowedOrigin(c.Request.Context(), origin)
		}
	}
	return cors.New(cfg)
}
