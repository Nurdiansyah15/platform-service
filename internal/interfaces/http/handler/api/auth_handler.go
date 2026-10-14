package api

import (
	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/auth"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/middleware"
	"platform-service/internal/interfaces/http/respond"
)

type AuthHandler struct {
	useCase *auth.AuthUseCase
}

func NewAuthHandler(useCase *auth.AuthUseCase) *AuthHandler {
	return &AuthHandler{useCase: useCase}
}

// Login godoc
//
//	@Summary	Login
//	@Tags		Auth
//	@Accept		json
//	@Produce	json
//	@Param		body		body		dto.LoginRequest	true	"Kredensial + code solution tujuan"
//	@Success	200			{object}	respond.SuccessBody{data=dto.AuthTokenResponse}
//	@Failure	401,403,422	{object}	respond.ErrorBody
//	@Router		/api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Login(c.Request.Context(), req, dto.ClientInfo{
		UserAgent: c.Request.UserAgent(),
		IPAddress: c.ClientIP(),
	})
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "logged in", resp)
}

// Refresh godoc
//
//	@Summary	Refresh token
//	@Tags		Auth
//	@Accept		json
//	@Produce	json
//	@Param		body	body		dto.RefreshTokenRequest	true	"Refresh token"
//	@Success	200		{object}	respond.SuccessBody{data=dto.AuthTokenResponse}
//	@Failure	401,422	{object}	respond.ErrorBody
//	@Router		/api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Refresh(c.Request.Context(), req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "token refreshed", resp)
}

// Logout godoc
//
//	@Summary	Logout (revoke sesi saat ini)
//	@Tags		Auth
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200		{object}	respond.SuccessBody
//	@Failure	401,403	{object}	respond.ErrorBody
//	@Router		/api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	actor, _ := middleware.PrincipalFrom(c)
	if err := h.useCase.Logout(c.Request.Context(), actor); err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "logged out", nil)
}

// Me godoc
//
//	@Summary	User & sesi saat ini
//	@Tags		Auth
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200		{object}	respond.SuccessBody{data=dto.MeResponse}
//	@Failure	401,403	{object}	respond.ErrorBody
//	@Router		/api/v1/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.Me(c.Request.Context(), actor)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "me", resp)
}

// Tenants godoc
//
//	@Summary		Tenants selectable for the session's solution
//	@Description	Tenant tempat user member active, tenant active, dan subscribe solution sesi (trialing / active).
//	@Tags			Auth
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200		{object}	respond.SuccessBody{data=[]dto.SelectableTenantResponse}
//	@Failure		401,403	{object}	respond.ErrorBody
//	@Router			/api/v1/auth/tenants [get]
func (h *AuthHandler) Tenants(c *gin.Context) {
	actor, _ := middleware.PrincipalFrom(c)
	items, err := h.useCase.Tenants(c.Request.Context(), actor)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "selectable tenants", items)
}

// SwitchTenant godoc
//
//	@Summary		Switch active tenant
//	@Description	Mengganti tenant aktif sesi dan menerbitkan access & refresh token baru; refresh token lama tidak berlaku.
//	@Tags			Auth
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body			body		dto.SwitchTenantRequest	true	"Tenant tujuan"
//	@Success		200				{object}	respond.SuccessBody{data=dto.AuthTokenResponse}
//	@Failure		401,403,422		{object}	respond.ErrorBody
//	@Router			/api/v1/auth/switch-tenant [post]
func (h *AuthHandler) SwitchTenant(c *gin.Context) {
	var req dto.SwitchTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.SwitchTenant(c.Request.Context(), actor, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "tenant switched", resp)
}
