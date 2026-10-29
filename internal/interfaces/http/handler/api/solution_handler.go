package api

import (
	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/solution"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/middleware"
	"platform-service/internal/interfaces/http/respond"
)

type SolutionHandler struct {
	useCase *solution.SolutionUseCase
}

func NewSolutionHandler(useCase *solution.SolutionUseCase) *SolutionHandler {
	return &SolutionHandler{useCase: useCase}
}

// Create godoc
//
//	@Summary	Create solution
//	@Tags		Solution Catalog
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		body	body		dto.CreateSolutionRequest	true	"Solution"
//	@Success	201		{object}	respond.SuccessBody{data=dto.SolutionResponse}
//	@Failure	401,403,409,422	{object}	respond.ErrorBody
//	@Router		/api/v1/solutions [post]
func (h *SolutionHandler) Create(c *gin.Context) {
	var req dto.CreateSolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Create(c.Request.Context(), req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "solution created", resp)
}

// List godoc
//
//	@Summary	List solutions
//	@Tags		Solution Catalog
//	@Security	BearerAuth
//	@Produce	json
//	@Param		status		query		string	false	"active | inactive (platform admin saja)"
//	@Param		search		query		string	false	"Cari di code / name"
//	@Param		page		query		int		false	"Halaman"
//	@Param		limit		query		int		false	"Jumlah per halaman (maks 100)"
//	@Param		sort_by		query		string	false	"code | name | created_at"
//	@Param		sort_type	query		string	false	"asc | desc"
//	@Success	200			{object}	respond.SuccessBody{data=[]dto.SolutionResponse,meta=dto.Meta}
//	@Failure	401,422		{object}	respond.ErrorBody
//	@Router		/api/v1/solutions [get]
func (h *SolutionHandler) List(c *gin.Context) {
	var req dto.ListSolutionQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, meta, err := h.useCase.List(c.Request.Context(), actor, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.SuccessWithMeta(c, 200, "solutions", items, meta)
}

// Get godoc
//
//	@Summary	Get solution
//	@Tags		Solution Catalog
//	@Security	BearerAuth
//	@Produce	json
//	@Param		solutionId	path		string	true	"Solution ID"
//	@Success	200			{object}	respond.SuccessBody{data=dto.SolutionResponse}
//	@Failure	401,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/solutions/{solutionId} [get]
func (h *SolutionHandler) Get(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.GetByID(c.Request.Context(), actor, param.SolutionID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "solution", resp)
}

// Update godoc
//
//	@Summary		Update solution
//	@Description	Ubah nama / deskripsi / status (kode tidak bisa diubah). Solution sesi admin sendiri tidak bisa dinonaktifkan.
//	@Tags			Solutions
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			solutionId			path		string						true	"Solution ID"
//	@Param			body				body		dto.UpdateSolutionRequest	true	"Solution"
//	@Success		200					{object}	respond.SuccessBody{data=dto.SolutionResponse}
//	@Failure		401,403,404,422		{object}	respond.ErrorBody
//	@Router			/api/v1/solutions/{solutionId} [patch]
func (h *SolutionHandler) Update(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.UpdateSolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.Update(c.Request.Context(), actor, param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "solution updated", resp)
}
