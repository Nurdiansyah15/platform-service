package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/system"
	"platform-service/internal/interfaces/http/respond"
)

type SystemHandler struct {
	useCase *system.SystemUseCase
}

func NewSystemHandler(useCase *system.SystemUseCase) *SystemHandler {
	return &SystemHandler{useCase: useCase}
}

// Health godoc
//
//	@Summary	Health check
//	@Tags		System
//	@Produce	json
//	@Success	200	{object}	respond.SuccessBody{data=dto.HealthResponse}
//	@Failure	503	{object}	respond.SuccessBody{data=dto.HealthResponse}
//	@Router		/healthz [get]
func (h *SystemHandler) Health(c *gin.Context) {
	resp := h.useCase.Health(c.Request.Context())
	status := http.StatusOK
	if resp.Status != dto.HealthStatusOK {
		status = http.StatusServiceUnavailable
	}
	respond.Success(c, status, "health check", resp)
}
