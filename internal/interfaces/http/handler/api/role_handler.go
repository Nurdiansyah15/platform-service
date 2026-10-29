package api

import (
	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/role"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/middleware"
	"platform-service/internal/interfaces/http/respond"
)

type RoleHandler struct {
	useCase *role.RoleUseCase
}

func NewRoleHandler(useCase *role.RoleUseCase) *RoleHandler {
	return &RoleHandler{useCase: useCase}
}

// ListGlobal godoc
//
//	@Summary	List global roles of a solution
//	@Tags		Roles
//	@Security	BearerAuth
//	@Produce	json
//	@Param		solutionId	path		string	true	"Solution ID"
//	@Param		search		query		string	false	"Cari di code / name"
//	@Param		page		query		int		false	"Halaman"
//	@Param		limit		query		int		false	"Jumlah per halaman (default 50, maks 200)"
//	@Success	200			{object}	respond.SuccessBody{data=[]dto.RoleResponse,meta=dto.Meta}
//	@Failure	401,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/solutions/{solutionId}/roles [get]
func (h *RoleHandler) ListGlobal(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ListRoleQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, meta, err := h.useCase.ListGlobal(c.Request.Context(), actor, param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.SuccessWithMeta(c, 200, "roles", items, meta)
}

// CreateGlobal godoc
//
//	@Summary	Create global role
//	@Tags		Roles
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		solutionId			path		string					true	"Solution ID"
//	@Param		body				body		dto.CreateRoleRequest	true	"Role"
//	@Success	201					{object}	respond.SuccessBody{data=dto.RoleResponse}
//	@Failure	401,403,404,409,422	{object}	respond.ErrorBody
//	@Router		/api/v1/solutions/{solutionId}/roles [post]
func (h *RoleHandler) CreateGlobal(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.CreateGlobal(c.Request.Context(), param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "role created", resp)
}

// ListPermissions godoc
//
//	@Summary		List role permissions
//	@Description	Custom role hanya terlihat oleh company pemiliknya (selain itu 404).
//	@Tags			Roles
//	@Security		BearerAuth
//	@Produce		json
//	@Param			roleId		path		string	true	"Role ID"
//	@Success		200			{object}	respond.SuccessBody{data=[]dto.RolePermissionResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/roles/{roleId}/permissions [get]
func (h *RoleHandler) ListPermissions(c *gin.Context) {
	var param dto.RoleIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, err := h.useCase.ListPermissions(c.Request.Context(), actor, param.RoleID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "role permissions", items)
}

// ReplaceGlobalPermissions godoc
//
//	@Summary		Replace global role permissions
//	@Description	Replace-all berdasarkan permission code. Role sistem tidak bisa diubah (403). Permission deprecated tidak bisa ditambahkan baru.
//	@Tags			Roles
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			roleId			path		string								true	"Role ID"
//	@Param			body			body		dto.ReplaceRolePermissionsRequest	true	"Permission codes"
//	@Success		200				{object}	respond.SuccessBody{data=[]dto.RolePermissionResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/roles/{roleId}/permissions [put]
func (h *RoleHandler) ReplaceGlobalPermissions(c *gin.Context) {
	var param dto.RoleIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ReplaceRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	items, err := h.useCase.ReplaceGlobalPermissions(c.Request.Context(), param.RoleID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "role permissions replaced", items)
}

// ListForCompany godoc
//
//	@Summary	List roles available to a company
//	@Description	Role global + custom role milik company untuk solution.
//	@Tags		Roles
//	@Security	BearerAuth
//	@Produce	json
//	@Param		companyId		path		string	true	"Company ID"
//	@Param		solutionId		path		string	true	"Solution ID"
//	@Param		search			query		string	false	"Cari di code / name"
//	@Param		page			query		int		false	"Halaman"
//	@Param		limit			query		int		false	"Jumlah per halaman (default 50, maks 200)"
//	@Success	200				{object}	respond.SuccessBody{data=[]dto.RoleResponse,meta=dto.Meta}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/companies/{companyId}/solutions/{solutionId}/roles [get]
func (h *RoleHandler) ListForCompany(c *gin.Context) {
	var param dto.CompanySolutionParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ListRoleQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, meta, err := h.useCase.ListForCompany(c.Request.Context(), actor, param.CompanyID, param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.SuccessWithMeta(c, 200, "roles", items, meta)
}

// CreateForCompany godoc
//
//	@Summary		Create company custom role
//	@Description	Company wajib subscribe solution (trialing / active).
//	@Tags			Roles
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			companyId			path		string					true	"Company ID"
//	@Param			solutionId			path		string					true	"Solution ID"
//	@Param			body				body		dto.CreateRoleRequest	true	"Role"
//	@Success		201					{object}	respond.SuccessBody{data=dto.RoleResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies/{companyId}/solutions/{solutionId}/roles [post]
func (h *RoleHandler) CreateForCompany(c *gin.Context) {
	var param dto.CompanySolutionParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.CreateForCompany(c.Request.Context(), actor, param.CompanyID, param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "role created", resp)
}

// ReplaceCompanyRolePermissions godoc
//
//	@Summary		Replace company custom role permissions
//	@Description	Hanya custom role milik company (role global / company lain → 404). Permission harus milik solution yang sama.
//	@Tags			Roles
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			companyId		path		string								true	"Company ID"
//	@Param			roleId			path		string								true	"Role ID"
//	@Param			body			body		dto.ReplaceRolePermissionsRequest	true	"Permission codes"
//	@Success		200				{object}	respond.SuccessBody{data=[]dto.RolePermissionResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies/{companyId}/roles/{roleId}/permissions [put]
func (h *RoleHandler) ReplaceCompanyRolePermissions(c *gin.Context) {
	var param dto.CompanyRoleParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ReplaceRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, err := h.useCase.ReplaceCompanyRolePermissions(c.Request.Context(), actor, param.CompanyID, param.RoleID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "role permissions replaced", items)
}

// GetGlobal godoc
//
//	@Summary	Get role
//	@Description	Detail role beserta jumlah permission & pemegang. Custom role hanya untuk company pemilik (selain itu 404).
//	@Tags		Roles
//	@Security	BearerAuth
//	@Produce	json
//	@Param		roleId			path		string	true	"Role ID"
//	@Success	200				{object}	respond.SuccessBody{data=dto.RoleResponse}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/roles/{roleId} [get]
func (h *RoleHandler) GetGlobal(c *gin.Context) {
	var param dto.RoleIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.GetGlobal(c.Request.Context(), actor, param.RoleID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "role", resp)
}

// UpdateGlobal godoc
//
//	@Summary		Update global role
//	@Description	Ubah nama / deskripsi. Role sistem tidak bisa diubah; custom role company → 404.
//	@Tags			Roles
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			roleId				path		string					true	"Role ID"
//	@Param			body				body		dto.UpdateRoleRequest	true	"Role"
//	@Success		200					{object}	respond.SuccessBody{data=dto.RoleResponse}
//	@Failure		401,403,404,422		{object}	respond.ErrorBody
//	@Router			/api/v1/roles/{roleId} [patch]
func (h *RoleHandler) UpdateGlobal(c *gin.Context) {
	var param dto.RoleIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.UpdateGlobal(c.Request.Context(), param.RoleID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "role updated", resp)
}

// DeleteGlobal godoc
//
//	@Summary		Delete global role
//	@Description	Soft delete; seluruh assignment role ini ikut dihapus. Role sistem tidak bisa dihapus.
//	@Tags			Roles
//	@Security		BearerAuth
//	@Produce		json
//	@Param			roleId			path		string	true	"Role ID"
//	@Success		200				{object}	respond.SuccessBody
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/roles/{roleId} [delete]
func (h *RoleHandler) DeleteGlobal(c *gin.Context) {
	var param dto.RoleIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	if err := h.useCase.DeleteGlobal(c.Request.Context(), param.RoleID); err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "role deleted", nil)
}

// GetForCompany godoc
//
//	@Summary	Get role in company context
//	@Description	Role global atau custom milik company; jumlah pemegang hanya dihitung di company ini.
//	@Tags		Roles
//	@Security	BearerAuth
//	@Produce	json
//	@Param		companyId		path		string	true	"Company ID"
//	@Param		roleId			path		string	true	"Role ID"
//	@Success	200				{object}	respond.SuccessBody{data=dto.RoleResponse}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/companies/{companyId}/roles/{roleId} [get]
func (h *RoleHandler) GetForCompany(c *gin.Context) {
	var param dto.CompanyRoleParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.GetForCompany(c.Request.Context(), actor, param.CompanyID, param.RoleID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "role", resp)
}

// ListHolders godoc
//
//	@Summary	List role holders in company
//	@Tags		Roles
//	@Security	BearerAuth
//	@Produce	json
//	@Param		companyId		path		string	true	"Company ID"
//	@Param		roleId			path		string	true	"Role ID"
//	@Success	200				{object}	respond.SuccessBody{data=[]dto.RoleHolderResponse}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/companies/{companyId}/roles/{roleId}/members [get]
func (h *RoleHandler) ListHolders(c *gin.Context) {
	var param dto.CompanyRoleParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, err := h.useCase.ListHolders(c.Request.Context(), actor, param.CompanyID, param.RoleID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "role holders", items)
}

// UpdateForCompany godoc
//
//	@Summary		Update company custom role
//	@Description	Ubah nama / deskripsi custom role milik company. Role global / company lain → 404.
//	@Tags			Roles
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			companyId		path		string					true	"Company ID"
//	@Param			roleId			path		string					true	"Role ID"
//	@Param			body			body		dto.UpdateRoleRequest	true	"Role"
//	@Success		200				{object}	respond.SuccessBody{data=dto.RoleResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies/{companyId}/roles/{roleId} [patch]
func (h *RoleHandler) UpdateForCompany(c *gin.Context) {
	var param dto.CompanyRoleParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.UpdateForCompany(c.Request.Context(), actor, param.CompanyID, param.RoleID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "role updated", resp)
}

// DeleteForCompany godoc
//
//	@Summary		Delete company custom role
//	@Description	Soft delete; assignment role ini ikut dihapus. Role global / company lain → 404.
//	@Tags			Roles
//	@Security		BearerAuth
//	@Produce		json
//	@Param			companyId		path		string	true	"Company ID"
//	@Param			roleId			path		string	true	"Role ID"
//	@Success		200				{object}	respond.SuccessBody
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies/{companyId}/roles/{roleId} [delete]
func (h *RoleHandler) DeleteForCompany(c *gin.Context) {
	var param dto.CompanyRoleParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	if err := h.useCase.DeleteForCompany(c.Request.Context(), actor, param.CompanyID, param.RoleID); err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "role deleted", nil)
}
