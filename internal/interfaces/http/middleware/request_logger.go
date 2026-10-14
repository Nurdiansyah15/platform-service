package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger adalah middleware Gin yang mencatat setiap request secara terstruktur.
//
// Level output:
//   - ERROR (msg: "server error") — status >= 500
//   - WARN  (msg: "client error") — status >= 400
//   - INFO  (msg: "request")      — status < 400
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path += "?" + raw
		}

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.String("latency", latency.String()),
			slog.String("ip", c.ClientIP()),
		}

		// Sertakan error yang di-handle oleh httperror.Handle (4xx/5xx).
		if rawErr, exists := c.Get("request_err"); exists {
			if err, ok := rawErr.(error); ok && err != nil {
				attrs = append(attrs, slog.String("error", err.Error()))
			}
		}
		if rawCode, exists := c.Get("request_err_code"); exists {
			if code, ok := rawCode.(string); ok && code != "" {
				attrs = append(attrs, slog.String("error_code", code))
			}
		}
		if rawMsg, exists := c.Get("request_err_message"); exists {
			if msg, ok := rawMsg.(string); ok && msg != "" {
				attrs = append(attrs, slog.String("error_message", msg))
			}
		}

		level := slog.LevelInfo
		msg := "request"
		if status >= 500 {
			level = slog.LevelError
			msg = "server error"
		} else if status >= 400 {
			level = slog.LevelWarn
			msg = "client error"
		}

		logger.LogAttrs(c.Request.Context(), level, msg, attrs...)
	}
}
