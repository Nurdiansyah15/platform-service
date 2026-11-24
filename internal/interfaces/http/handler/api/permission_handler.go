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
//	@Description	Dipanggil solution saat deploy. Idempotent per version: permission baru dibuat, yang ada diperbarui / diaktifkan ulang, yang hilang menjadi deprecated (tidak dihapus). `scope_types` (scope custom) dan `groups` (pengelompokan tampilan) opsional dan diperlakukan sama.
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
//	@Param		group		query		string	false	"Code group; '-' = tanpa grup"
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

// ListGroups godoc
//
//	@Summary	List permission groups of a solution
//	@Tags		Permissions
//	@Security	BearerAuth
//	@Produce	json
//	@Param		solutionId	path		string	true	"Solution ID"
//	@Param		status		query		string	false	"active | deprecated"
//	@Success	200			{object}	respond.SuccessBody{data=[]dto.PermissionGroupResponse}
//	@Failure	401,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/solutions/{solutionId}/permission-groups [get]
func (h *PermissionHandler) ListGroups(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ListCatalogQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, err := h.useCase.ListGroups(c.Request.Context(), actor, param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "permission groups", items)
}

// ListScopeTypes godoc
//
//	@Summary		List scope types of a solution
//	@Description	Scope yang bisa dipakai role assignment solution: company & branch (built-in) lalu scope type custom dari katalog.
//	@Tags			Permissions
//	@Security		BearerAuth
//	@Produce		json
//	@Param			solutionId	path		string	true	"Solution ID"
//	@Param			status		query		string	false	"active | deprecated"
//	@Success		200			{object}	respond.SuccessBody{data=[]dto.ScopeTypeResponse}
//	@Failure		401,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/solutions/{solutionId}/scope-types [get]
func (h *PermissionHandler) ListScopeTypes(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ListCatalogQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, err := h.useCase.ListScopeTypes(c.Request.Context(), actor, param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "scope types", items)
}

// Create godoc
//
//	@Summary		Create a manual permission
//	@Description	Permission manual dikelola platform admin dan tidak disentuh sync katalog. Sync yang memuat code permission manual ditolak (409).
//	@Tags			Permissions
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			solutionId			path		string						true	"Solution ID"
//	@Param			body				body		dto.CreatePermissionRequest	true	"Permission"
//	@Success		201					{object}	respond.SuccessBody{data=dto.PermissionResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/solutions/{solutionId}/permissions [post]
func (h *PermissionHandler) Create(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Create(c.Request.Context(), param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "permission created", resp)
}

// Get godoc
//
//	@Summary	Get a permission
//	@Tags		Permissions
//	@Security	BearerAuth
//	@Produce	json
//	@Param		permissionId	path		string	true	"Permission ID"
//	@Success	200				{object}	respond.SuccessBody{data=dto.PermissionResponse}
//	@Failure	401,404,422		{object}	respond.ErrorBody
//	@Router		/api/v1/permissions/{permissionId} [get]
func (h *PermissionHandler) Get(c *gin.Context) {
	var param dto.PermissionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.Get(c.Request.Context(), actor, param.PermissionID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "permission", resp)
}

// Update godoc
//
//	@Summary		Update a manual permission
//	@Description	Hanya permission manual (hasil sync → 409). Code tidak bisa diubah. `status: deprecated` = hapus (permission tidak bisa ditambahkan ke role baru; yang sudah terpasang tetap berlaku).
//	@Tags			Permissions
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			permissionId		path		string						true	"Permission ID"
//	@Param			body				body		dto.UpdatePermissionRequest	true	"Field yang diubah"
//	@Success		200					{object}	respond.SuccessBody{data=dto.PermissionResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/permissions/{permissionId} [patch]
func (h *PermissionHandler) Update(c *gin.Context) {
	var param dto.PermissionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.UpdatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Update(c.Request.Context(), param.PermissionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "permission updated", resp)
}

// CreateGroup godoc
//
//	@Summary		Create a manual permission group
//	@Description	Group manual dikelola platform admin dan tidak disentuh sync katalog. Sync yang memuat code group manual ditolak (409).
//	@Tags			Permissions
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			solutionId			path		string								true	"Solution ID"
//	@Param			body				body		dto.CreatePermissionGroupRequest	true	"Group"
//	@Success		201					{object}	respond.SuccessBody{data=dto.PermissionGroupResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/solutions/{solutionId}/permission-groups [post]
func (h *PermissionHandler) CreateGroup(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.CreatePermissionGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.CreateGroup(c.Request.Context(), param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "permission group created", resp)
}

// GetGroup godoc
//
//	@Summary	Get a permission group
//	@Tags		Permissions
//	@Security	BearerAuth
//	@Produce	json
//	@Param		groupId		path		string	true	"Group ID"
//	@Success	200			{object}	respond.SuccessBody{data=dto.PermissionGroupResponse}
//	@Failure	401,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/permission-groups/{groupId} [get]
func (h *PermissionHandler) GetGroup(c *gin.Context) {
	var param dto.PermissionGroupIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.GetGroup(c.Request.Context(), actor, param.GroupID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "permission group", resp)
}

// UpdateGroup godoc
//
//	@Summary		Update a manual permission group
//	@Description	Hanya group manual (hasil sync → 409). Code tidak bisa diubah. `status: deprecated` = hapus (tidak bisa dipilih untuk permission baru; permission yang sudah memakainya tetap di grup itu).
//	@Tags			Permissions
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			groupId				path		string								true	"Group ID"
//	@Param			body				body		dto.UpdatePermissionGroupRequest	true	"Field yang diubah"
//	@Success		200					{object}	respond.SuccessBody{data=dto.PermissionGroupResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/permission-groups/{groupId} [patch]
func (h *PermissionHandler) UpdateGroup(c *gin.Context) {
	var param dto.PermissionGroupIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.UpdatePermissionGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.UpdateGroup(c.Request.Context(), param.GroupID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "permission group updated", resp)
}

// CreateScopeType godoc
//
//	@Summary		Create a manual scope type
//	@Description	Scope type custom manual dikelola platform admin dan tidak disentuh sync katalog. Sync yang memuat code-nya ditolak (409). Code company / branch reserved.
//	@Tags			Permissions
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			solutionId			path		string						true	"Solution ID"
//	@Param			body				body		dto.CreateScopeTypeRequest	true	"Scope type"
//	@Success		201					{object}	respond.SuccessBody{data=dto.ScopeTypeResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/solutions/{solutionId}/scope-types [post]
func (h *PermissionHandler) CreateScopeType(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.CreateScopeTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.CreateScopeType(c.Request.Context(), param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "scope type created", resp)
}

// GetScopeType godoc
//
//	@Summary	Get a custom scope type
//	@Tags		Permissions
//	@Security	BearerAuth
//	@Produce	json
//	@Param		scopeTypeId	path		string	true	"Scope type ID"
//	@Success	200			{object}	respond.SuccessBody{data=dto.ScopeTypeResponse}
//	@Failure	401,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/scope-types/{scopeTypeId} [get]
func (h *PermissionHandler) GetScopeType(c *gin.Context) {
	var param dto.ScopeTypeIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.GetScopeType(c.Request.Context(), actor, param.ScopeTypeID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "scope type", resp)
}

// UpdateScopeType godoc
//
//	@Summary		Update a manual scope type
//	@Description	Hanya scope type manual (hasil sync → 409). Code tidak bisa diubah. `status: deprecated` = hapus (tidak bisa dipakai assignment baru; assignment lama tetap berlaku).
//	@Tags			Permissions
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			scopeTypeId			path		string						true	"Scope type ID"
//	@Param			body				body		dto.UpdateScopeTypeRequest	true	"Field yang diubah"
//	@Success		200					{object}	respond.SuccessBody{data=dto.ScopeTypeResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/scope-types/{scopeTypeId} [patch]
func (h *PermissionHandler) UpdateScopeType(c *gin.Context) {
	var param dto.ScopeTypeIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.UpdateScopeTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.UpdateScopeType(c.Request.Context(), param.ScopeTypeID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "scope type updated", resp)
}
