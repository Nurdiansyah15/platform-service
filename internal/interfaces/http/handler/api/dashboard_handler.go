package api

import (
	"github.com/gin-gonic/gin"

	"platform-service/internal/app/usecase/dashboard"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/respond"
)

type DashboardHandler struct {
	useCase *dashboard.DashboardUseCase
}

func NewDashboardHandler(useCase *dashboard.DashboardUseCase) *DashboardHandler {
	return &DashboardHandler{useCase: useCase}
}

// AdminSummary godoc
//
//	@Summary	Platform summary
//	@Description	Jumlah user, company, solution, dan client aktif untuk dashboard admin.
//	@Tags		Dashboard
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200		{object}	respond.SuccessBody{data=dto.AdminSummaryResponse}
//	@Failure	401,403	{object}	respond.ErrorBody
//	@Router		/api/v1/admin/summary [get]
func (h *DashboardHandler) AdminSummary(c *gin.Context) {
	resp, err := h.useCase.AdminSummary(c.Request.Context())
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "admin summary", resp)
}
