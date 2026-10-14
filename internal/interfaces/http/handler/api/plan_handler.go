package api

import (
	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/plan"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/middleware"
	"platform-service/internal/interfaces/http/respond"
)

type PlanHandler struct {
	useCase *plan.PlanUseCase
}

func NewPlanHandler(useCase *plan.PlanUseCase) *PlanHandler {
	return &PlanHandler{useCase: useCase}
}

// Create godoc
//
//	@Summary		Create plan
//	@Description	price boleh angka atau string angka (maks 2 desimal); respons selalu string (mis. "150000.00").
//	@Tags			Plans
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			solutionId			path		string					true	"Solution ID"
//	@Param			body				body		dto.CreatePlanRequest	true	"Plan"
//	@Success		201					{object}	respond.SuccessBody{data=dto.PlanResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/solutions/{solutionId}/plans [post]
func (h *PlanHandler) Create(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Create(c.Request.Context(), param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "plan created", resp)
}

// List godoc
//
//	@Summary		List plans of a solution
//	@Description	Non-admin hanya melihat plan aktif dari solution aktif.
//	@Tags			Plans
//	@Security		BearerAuth
//	@Produce		json
//	@Param			solutionId		path		string	true	"Solution ID"
//	@Param			is_active		query		bool	false	"Filter (platform admin saja)"
//	@Param			billing_cycle	query		string	false	"monthly | yearly"
//	@Param			search			query		string	false	"Cari di code / name"
//	@Param			page			query		int		false	"Halaman"
//	@Param			limit			query		int		false	"Jumlah per halaman (maks 100)"
//	@Param			sort_by			query		string	false	"price | code | name | created_at"
//	@Param			sort_type		query		string	false	"asc | desc"
//	@Success		200				{object}	respond.SuccessBody{data=[]dto.PlanResponse,meta=dto.Meta}
//	@Failure		401,404,422		{object}	respond.ErrorBody
//	@Router			/api/v1/solutions/{solutionId}/plans [get]
func (h *PlanHandler) List(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ListPlanQuery
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
	respond.SuccessWithMeta(c, 200, "plans", items, meta)
}

// ListBenefits godoc
//
//	@Summary	List plan benefits
//	@Tags		Plans
//	@Security	BearerAuth
//	@Produce	json
//	@Param		planId		path		string	true	"Plan ID"
//	@Success	200			{object}	respond.SuccessBody{data=[]dto.AssignedBenefitResponse}
//	@Failure	401,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/plans/{planId}/benefits [get]
func (h *PlanHandler) ListBenefits(c *gin.Context) {
	var param dto.PlanIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, err := h.useCase.ListBenefits(c.Request.Context(), actor, param.PlanID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "plan benefits", items)
}

// ReplaceBenefits godoc
//
//	@Summary		Replace plan benefits
//	@Description	Replace-all: daftar ini menjadi seluruh benefit plan (array kosong = hapus semua). Tidak memengaruhi subscription berjalan.
//	@Tags			Plans
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			planId			path		string						true	"Plan ID"
//	@Param			body			body		dto.ReplaceBenefitsRequest	true	"Benefit values"
//	@Success		200				{object}	respond.SuccessBody{data=[]dto.AssignedBenefitResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/plans/{planId}/benefits [put]
func (h *PlanHandler) ReplaceBenefits(c *gin.Context) {
	var param dto.PlanIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ReplaceBenefitsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	items, err := h.useCase.ReplaceBenefits(c.Request.Context(), param.PlanID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "plan benefits replaced", items)
}
