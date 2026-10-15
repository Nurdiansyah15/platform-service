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
//	@Description	Custom role hanya terlihat oleh tenant pemiliknya (selain itu 404).
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

// ListForTenant godoc
//
//	@Summary	List roles available to a tenant
//	@Description	Role global + custom role milik tenant untuk solution.
//	@Tags		Roles
//	@Security	BearerAuth
//	@Produce	json
//	@Param		tenantId		path		string	true	"Tenant ID"
//	@Param		solutionId		path		string	true	"Solution ID"
//	@Param		search			query		string	false	"Cari di code / name"
//	@Param		page			query		int		false	"Halaman"
//	@Param		limit			query		int		false	"Jumlah per halaman (default 50, maks 200)"
//	@Success	200				{object}	respond.SuccessBody{data=[]dto.RoleResponse,meta=dto.Meta}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/tenants/{tenantId}/solutions/{solutionId}/roles [get]
func (h *RoleHandler) ListForTenant(c *gin.Context) {
	var param dto.TenantSolutionParam
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
	items, meta, err := h.useCase.ListForTenant(c.Request.Context(), actor, param.TenantID, param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.SuccessWithMeta(c, 200, "roles", items, meta)
}

// CreateForTenant godoc
//
//	@Summary		Create tenant custom role
//	@Description	Tenant wajib subscribe solution (trialing / active).
//	@Tags			Roles
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			tenantId			path		string					true	"Tenant ID"
//	@Param			solutionId			path		string					true	"Solution ID"
//	@Param			body				body		dto.CreateRoleRequest	true	"Role"
//	@Success		201					{object}	respond.SuccessBody{data=dto.RoleResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/tenants/{tenantId}/solutions/{solutionId}/roles [post]
func (h *RoleHandler) CreateForTenant(c *gin.Context) {
	var param dto.TenantSolutionParam
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
	resp, err := h.useCase.CreateForTenant(c.Request.Context(), actor, param.TenantID, param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "role created", resp)
}

// ReplaceTenantRolePermissions godoc
//
//	@Summary		Replace tenant custom role permissions
//	@Description	Hanya custom role milik tenant (role global / tenant lain → 404). Permission harus milik solution yang sama.
//	@Tags			Roles
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			tenantId		path		string								true	"Tenant ID"
//	@Param			roleId			path		string								true	"Role ID"
//	@Param			body			body		dto.ReplaceRolePermissionsRequest	true	"Permission codes"
//	@Success		200				{object}	respond.SuccessBody{data=[]dto.RolePermissionResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/tenants/{tenantId}/roles/{roleId}/permissions [put]
func (h *RoleHandler) ReplaceTenantRolePermissions(c *gin.Context) {
	var param dto.TenantRoleParam
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
	items, err := h.useCase.ReplaceTenantRolePermissions(c.Request.Context(), actor, param.TenantID, param.RoleID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "role permissions replaced", items)
}
