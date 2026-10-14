package api

import (
	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/permission"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/middleware"
	"platform-service/internal/interfaces/http/respond"
)

type PermissionHandler struct {
	useCase *permission.PermissionUseCase
}

func NewPermissionHandler(useCase *permission.PermissionUseCase) *PermissionHandler {
	return &PermissionHandler{useCase: useCase}
}

// Sync godoc
//
//	@Summary		Sync permission catalog
//	@Description	Dipanggil solution saat deploy. Idempotent per version: permission baru dibuat, yang ada diperbarui / diaktifkan ulang, yang hilang menjadi deprecated (tidak dihapus).
//	@Tags			Permissions
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			solutionId			path		string						true	"Solution ID"
//	@Param			body				body		dto.SyncPermissionsRequest	true	"Katalog lengkap"
//	@Success		200					{object}	respond.SuccessBody{data=dto.SyncPermissionsResponse}
//	@Failure		401,403,404,422		{object}	respond.ErrorBody
//	@Router			/api/v1/solutions/{solutionId}/permissions/sync [put]
func (h *PermissionHandler) Sync(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.SyncPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.Sync(c.Request.Context(), actor, param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "permission catalog synced", resp)
}

// List godoc
//
//	@Summary	List permissions of a solution
//	@Tags		Permissions
//	@Security	BearerAuth
//	@Produce	json
//	@Param		solutionId	path		string	true	"Solution ID"
//	@Param		status		query		string	false	"active | deprecated"
//	@Param		search		query		string	false	"Cari di code / name"
//	@Param		page		query		int		false	"Halaman"
//	@Param		limit		query		int		false	"Jumlah per halaman (default 50, maks 200)"
//	@Param		sort_by		query		string	false	"code | name | created_at"
//	@Param		sort_type	query		string	false	"asc | desc"
//	@Success	200			{object}	respond.SuccessBody{data=[]dto.PermissionResponse,meta=dto.Meta}
//	@Failure	401,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/solutions/{solutionId}/permissions [get]
func (h *PermissionHandler) List(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ListPermissionQuery
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
	respond.SuccessWithMeta(c, 200, "permissions", items, meta)
}
