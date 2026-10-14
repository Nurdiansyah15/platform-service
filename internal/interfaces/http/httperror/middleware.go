package httperror

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	appErr "platform-service/internal/app/apperror"
)

// Middleware memulihkan panic menjadi 500 dan menangani error yang dititipkan via c.Error().
func Middleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.ErrorContext(c.Request.Context(), "panic recovered",
					slog.Any("recovered", rec),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
				)
				Handle(c, internalError(string(appErr.CodeInternal)))
			}
		}()

		c.Next()

		if len(c.Errors) > 0 && !c.Writer.Written() {
			Handle(c, c.Errors.Last().Err)
		}
	}
}

// NoRoute menghasilkan 404 ber-envelope untuk route yang tidak terdaftar.
func NoRoute(c *gin.Context) {
	Handle(c, appErr.RouteNotFound())
}
