package middleware

import (
	"platform-service/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID adalah middleware Gin yang menandai tiap request dengan ID unik
// untuk korelasi log (request_id).
//
// Server selalu generate ID sendiri — header X-Request-Id dari client TIDAK
// pernah dipakai sebagai request_id internal (menghindari spoofing/collision).
// Kalau client mengirim X-Request-Id, nilainya di-echo apa adanya lewat
// response header terpisah X-Client-Request-Id, murni untuk kenyamanan client
// mencocokkan ID mereka sendiri.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := uuid.NewString()

		if clientID := c.GetHeader("X-Request-Id"); clientID != "" {
			c.Writer.Header().Set("X-Client-Request-Id", clientID)
		}

		c.Set("request_id", id)
		c.Writer.Header().Set("X-Request-Id", id)
		c.Request = c.Request.WithContext(logger.WithRequestID(c.Request.Context(), id))

		c.Next()
	}
}
