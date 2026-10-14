package api

import (
	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/benefit"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/middleware"
	"platform-service/internal/interfaces/http/respond"
)

type BenefitHandler struct {
	useCase *benefit.BenefitUseCase
}

func NewBenefitHandler(useCase *benefit.BenefitUseCase) *BenefitHandler {
	return &BenefitHandler{useCase: useCase}
}

// Create godoc
//
//	@Summary		Create benefit
//	@Description	Kombinasi valid — boolean: any_true/override; integer & decimal: sum/max/min/override (metering none/counter/gauge); string & json: override. usage_reset selain never hanya untuk counter.
//	@Tags			Benefits
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			solutionId			path		string						true	"Solution ID"
//	@Param			body				body		dto.CreateBenefitRequest	true	"Benefit"
//	@Success		201					{object}	respond.SuccessBody{data=dto.BenefitResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/solutions/{solutionId}/benefits [post]
func (h *BenefitHandler) Create(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.CreateBenefitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Create(c.Request.Context(), param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "benefit created", resp)
}

// List godoc
//
//	@Summary	List benefits of a solution
//	@Tags		Benefits
//	@Security	BearerAuth
//	@Produce	json
//	@Param		solutionId		path		string	true	"Solution ID"
//	@Param		value_type		query		string	false	"boolean | integer | decimal | string | json"
//	@Param		metering		query		string	false	"none | counter | gauge"
//	@Param		search			query		string	false	"Cari di code / name"
//	@Param		page			query		int		false	"Halaman"
//	@Param		limit			query		int		false	"Jumlah per halaman (default 50, maks 200)"
//	@Param		sort_by			query		string	false	"code | name | created_at"
//	@Param		sort_type		query		string	false	"asc | desc"
//	@Success	200				{object}	respond.SuccessBody{data=[]dto.BenefitResponse,meta=dto.Meta}
//	@Failure	401,404,422		{object}	respond.ErrorBody
//	@Router		/api/v1/solutions/{solutionId}/benefits [get]
func (h *BenefitHandler) List(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ListBenefitQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, meta, err := h.useCase.List(c.Request.Context(), actor, param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.SuccessWithMeta(c, 200, "benefits", items, meta)
}

// Get godoc
//
//	@Summary	Get benefit
//	@Tags		Benefits
//	@Security	BearerAuth
//	@Produce	json
//	@Param		benefitId	path		string	true	"Benefit ID"
//	@Success	200			{object}	respond.SuccessBody{data=dto.BenefitResponse}
//	@Failure	401,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/benefits/{benefitId} [get]
func (h *BenefitHandler) Get(c *gin.Context) {
	var param dto.BenefitIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.Get(c.Request.Context(), actor, param.BenefitID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "benefit", resp)
}
