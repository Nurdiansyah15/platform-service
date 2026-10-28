package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/usage"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/middleware"
	"platform-service/internal/interfaces/http/respond"
)

type UsageHandler struct {
	useCase *usage.UsageUseCase
}

func NewUsageHandler(useCase *usage.UsageUseCase) *UsageHandler {
	return &UsageHandler{useCase: useCase}
}

// ReportDelta godoc
//
//	@Summary		Report counter usage (delta)
//	@Description	used = used + quantity untuk benefit metering counter. Idempotent per idempotency_key: 201 = diterapkan, 200 = replay (tidak diterapkan ulang).
//	@Tags			Usage
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			companyId			path		string							true	"Company ID"
//	@Param			solutionId			path		string							true	"Solution ID (harus solution client)"
//	@Param			body				body		dto.ReportUsageDeltaRequest		true	"Laporan"
//	@Success		201					{object}	respond.SuccessBody{data=dto.UsageReportResponse}
//	@Success		200					{object}	respond.SuccessBody{data=dto.UsageReportResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies/{companyId}/solutions/{solutionId}/usage [post]
func (h *UsageHandler) ReportDelta(c *gin.Context) {
	var param dto.CompanySolutionParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ReportUsageDeltaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.ReportDelta(c.Request.Context(), actor, param.CompanyID, param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	status := http.StatusCreated
	if resp.Replayed {
		status = http.StatusOK
	}
	respond.Success(c, status, "usage recorded", resp)
}

// ReportAbsolute godoc
//
//	@Summary		Report gauge usage (absolute)
//	@Description	used = quantity untuk benefit metering gauge. Laporan dengan occurred_at lebih lama dari yang sudah diterapkan diabaikan. Idempotent per idempotency_key.
//	@Tags			Usage
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			companyId			path		string							true	"Company ID"
//	@Param			solutionId			path		string							true	"Solution ID (harus solution client)"
//	@Param			benefitCode			path		string							true	"Benefit code"
//	@Param			body				body		dto.ReportUsageAbsoluteRequest	true	"Laporan"
//	@Success		200					{object}	respond.SuccessBody{data=dto.UsageReportResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies/{companyId}/solutions/{solutionId}/usage/{benefitCode} [put]
func (h *UsageHandler) ReportAbsolute(c *gin.Context) {
	var param dto.CompanySolutionBenefitParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ReportUsageAbsoluteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.ReportAbsolute(c.Request.Context(), actor, param.CompanyID, param.SolutionID, param.BenefitCode, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "usage recorded", resp)
}
