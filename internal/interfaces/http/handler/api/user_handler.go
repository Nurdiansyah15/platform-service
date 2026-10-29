package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/user"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/middleware"
	"platform-service/internal/interfaces/http/respond"
)

type UserHandler struct {
	useCase *user.UserUseCase
}

func NewUserHandler(useCase *user.UserUseCase) *UserHandler {
	return &UserHandler{useCase: useCase}
}

// Create godoc
//
//	@Summary	Create user
//	@Tags		Users
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		body			body		dto.CreateUserRequest	true	"User"
//	@Success	201				{object}	respond.SuccessBody{data=dto.UserResponse}
//	@Failure	401,403,409,422	{object}	respond.ErrorBody
//	@Router		/api/v1/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Create(c.Request.Context(), req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "user created", resp)
}

// List godoc
//
//	@Summary	List users
//	@Tags		Users
//	@Security	BearerAuth
//	@Produce	json
//	@Param		status				query		string	false	"active | inactive | suspended"
//	@Param		is_platform_admin	query		bool	false	"Filter platform admin"
//	@Param		search				query		string	false	"Cari di email / nama"
//	@Param		page				query		int		false	"Halaman"
//	@Param		limit				query		int		false	"Jumlah per halaman (maks 100)"
//	@Param		sort_by				query		string	false	"email | name | created_at"
//	@Param		sort_type			query		string	false	"asc | desc"
//	@Success	200					{object}	respond.SuccessBody{data=[]dto.UserResponse,meta=dto.Meta}
//	@Failure	401,403,422			{object}	respond.ErrorBody
//	@Router		/api/v1/users [get]
func (h *UserHandler) List(c *gin.Context) {
	var req dto.ListUserQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	items, meta, err := h.useCase.List(c.Request.Context(), req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.SuccessWithMeta(c, http.StatusOK, "users", items, meta)
}

// Get godoc
//
//	@Summary	Get user
//	@Description	User beserta keanggotaan company-nya.
//	@Tags		Users
//	@Security	BearerAuth
//	@Produce	json
//	@Param		userId			path		string	true	"User ID"
//	@Success	200				{object}	respond.SuccessBody{data=dto.UserDetailResponse}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/users/{userId} [get]
func (h *UserHandler) Get(c *gin.Context) {
	var param dto.UserIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Get(c.Request.Context(), param.UserID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "user", resp)
}

// Update godoc
//
//	@Summary		Update user
//	@Description	Ubah email / nama / flag platform admin, atau reset password. Admin tidak bisa mencabut flag admin dirinya sendiri.
//	@Tags			Users
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			userId				path		string					true	"User ID"
//	@Param			body				body		dto.UpdateUserRequest	true	"User"
//	@Success		200					{object}	respond.SuccessBody{data=dto.UserResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/users/{userId} [patch]
func (h *UserHandler) Update(c *gin.Context) {
	var param dto.UserIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.Update(c.Request.Context(), actor, param.UserID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "user updated", resp)
}

// ChangeStatus godoc
//
//	@Summary		Change user status
//	@Description	User yang tidak active langsung tidak bisa memakai token-nya. Admin tidak bisa menonaktifkan dirinya sendiri.
//	@Tags			Users
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			userId			path		string						true	"User ID"
//	@Param			body			body		dto.UpdateUserStatusRequest	true	"Status"
//	@Success		200				{object}	respond.SuccessBody{data=dto.UserResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/users/{userId}/status [patch]
func (h *UserHandler) ChangeStatus(c *gin.Context) {
	var param dto.UserIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.ChangeStatus(c.Request.Context(), actor, param.UserID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "user status updated", resp)
}

// Me godoc
//
//	@Summary	Get my profile
//	@Tags		Users
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200		{object}	respond.SuccessBody{data=dto.UserResponse}
//	@Failure	401,403	{object}	respond.ErrorBody
//	@Router		/api/v1/users/me [get]
func (h *UserHandler) Me(c *gin.Context) {
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.Me(c.Request.Context(), actor)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "me", resp)
}

// UpdateMe godoc
//
//	@Summary		Update my profile
//	@Description	Ubah nama dan/atau password. Ganti password wajib menyertakan current_password.
//	@Tags			Users
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body		body		dto.UpdateMeRequest	true	"Profil"
//	@Success		200			{object}	respond.SuccessBody{data=dto.UserResponse}
//	@Failure		401,403,422	{object}	respond.ErrorBody
//	@Router			/api/v1/users/me [patch]
func (h *UserHandler) UpdateMe(c *gin.Context) {
	var req dto.UpdateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.UpdateMe(c.Request.Context(), actor, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "profile updated", resp)
}
