package api

import (
	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/addon"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/middleware"
	"platform-service/internal/interfaces/http/respond"
)

type AddonHandler struct {
	useCase *addon.AddonUseCase
}

func NewAddonHandler(useCase *addon.AddonUseCase) *AddonHandler {
	return &AddonHandler{useCase: useCase}
}

// Create godoc
//
//	@Summary		Create add-on
//	@Description	price boleh angka atau string angka (maks 2 desimal); respons selalu string (mis. "25000.00").
//	@Tags			Add-ons
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			solutionId			path		string					true	"Solution ID"
//	@Param			body				body		dto.CreateAddonRequest	true	"Add-on"
//	@Success		201					{object}	respond.SuccessBody{data=dto.AddonResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/solutions/{solutionId}/addons [post]
func (h *AddonHandler) Create(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.CreateAddonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Create(c.Request.Context(), param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "add-on created", resp)
}

// List godoc
//
//	@Summary		List add-ons of a solution
//	@Description	Non-admin hanya melihat add-on aktif dari solution aktif.
//	@Tags			Add-ons
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
//	@Success		200				{object}	respond.SuccessBody{data=[]dto.AddonResponse,meta=dto.Meta}
//	@Failure		401,404,422		{object}	respond.ErrorBody
//	@Router			/api/v1/solutions/{solutionId}/addons [get]
func (h *AddonHandler) List(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ListAddonQuery
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
	respond.SuccessWithMeta(c, 200, "add-ons", items, meta)
}

// ListBenefits godoc
//
//	@Summary	List add-on benefits
//	@Tags		Add-ons
//	@Security	BearerAuth
//	@Produce	json
//	@Param		addonId		path		string	true	"Add-on ID"
//	@Success	200			{object}	respond.SuccessBody{data=[]dto.AssignedBenefitResponse}
//	@Failure	401,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/addons/{addonId}/benefits [get]
func (h *AddonHandler) ListBenefits(c *gin.Context) {
	var param dto.AddonIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, err := h.useCase.ListBenefits(c.Request.Context(), actor, param.AddonID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "add-on benefits", items)
}

// ReplaceBenefits godoc
//
//	@Summary		Replace add-on benefits
//	@Description	Replace-all: daftar ini menjadi seluruh benefit add-on (array kosong = hapus semua). Tidak memengaruhi subscription berjalan.
//	@Tags			Add-ons
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			addonId			path		string						true	"Add-on ID"
//	@Param			body			body		dto.ReplaceBenefitsRequest	true	"Benefit values"
//	@Success		200				{object}	respond.SuccessBody{data=[]dto.AssignedBenefitResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/addons/{addonId}/benefits [put]
func (h *AddonHandler) ReplaceBenefits(c *gin.Context) {
	var param dto.AddonIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ReplaceBenefitsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	items, err := h.useCase.ReplaceBenefits(c.Request.Context(), param.AddonID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "add-on benefits replaced", items)
}
