package router

import (
	"github.com/gin-gonic/gin"

	"platform-service/internal/interfaces/http/handler/api"
)

// RegisterSystemRoutes mendaftarkan endpoint operasional di luar /api/v1.
func RegisterSystemRoutes(r *gin.Engine, h *api.SystemHandler) {
	r.GET("/healthz", h.Health)
}
