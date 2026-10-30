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

// Companies godoc
//
//	@Summary		Companies selectable for the session's solution
//	@Description	Company tempat user member active, company active, dan subscribe solution sesi (trialing / active).
//	@Tags			Auth
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200		{object}	respond.SuccessBody{data=[]dto.SelectableCompanyResponse}
//	@Failure		401,403	{object}	respond.ErrorBody
//	@Router			/api/v1/auth/companies [get]
func (h *AuthHandler) Companies(c *gin.Context) {
	actor, _ := middleware.PrincipalFrom(c)
	items, err := h.useCase.Companies(c.Request.Context(), actor)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "selectable companies", items)
}

// SwitchCompany godoc
//
//	@Summary		Switch active company
//	@Description	Mengganti company aktif sesi dan menerbitkan access & refresh token baru; refresh token lama tidak berlaku.
//	@Tags			Auth
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body			body		dto.SwitchCompanyRequest	true	"Company tujuan"
//	@Success		200				{object}	respond.SuccessBody{data=dto.AuthTokenResponse}
//	@Failure		401,403,422		{object}	respond.ErrorBody
//	@Router			/api/v1/auth/switch-company [post]
func (h *AuthHandler) SwitchCompany(c *gin.Context) {
	var req dto.SwitchCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.SwitchCompany(c.Request.Context(), actor, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "company switched", resp)
}

// Authorize godoc
//
//	@Summary		SSO: issue authorization code
//	@Description	Langkah 1 SSO console → solution (authorization code + PKCE). Hanya untuk token sesi console.
//	@Description	Kode berlaku 60 detik dan sekali pakai; redirect_uri wajib terdaftar persis di solution tujuan.
//	@Tags			Auth
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body			body		dto.AuthorizeRequest	true	"Solution tujuan, redirect_uri, PKCE"
//	@Success		200				{object}	respond.SuccessBody{data=dto.AuthorizeResponse}
//	@Failure		401,403,422		{object}	respond.ErrorBody
//	@Router			/api/v1/auth/authorize [post]
func (h *AuthHandler) Authorize(c *gin.Context) {
	var req dto.AuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.Authorize(c.Request.Context(), actor, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "authorization code issued", resp)
}

// TokenExchange godoc
//
//	@Summary		SSO: exchange authorization code
//	@Description	Langkah 2 SSO: solution menukar code + code_verifier menjadi sesi baru (respons sama dengan login).
//	@Description	Semua kegagalan (kode tidak ada / dipakai / kedaluwarsa, PKCE atau redirect_uri salah) → 400 DOMAIN_AUTH_INVALID_GRANT.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body		body		dto.TokenExchangeRequest	true	"Code, code_verifier, redirect_uri"
//	@Success		200			{object}	respond.SuccessBody{data=dto.AuthTokenResponse}
//	@Failure		400,422		{object}	respond.ErrorBody
//	@Router			/api/v1/auth/token-exchange [post]
func (h *AuthHandler) TokenExchange(c *gin.Context) {
	var req dto.TokenExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.ExchangeCode(c.Request.Context(), req, dto.ClientInfo{
		UserAgent: c.Request.UserAgent(),
		IPAddress: c.ClientIP(),
	})
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "logged in", resp)
}

// Apps godoc
//
//	@Summary		Apps available to the active company
//	@Description	Shortcut solution untuk launcher console dan menu apps di navbar solution. Token user dari solution mana pun.
//	@Description	Tanpa company aktif → daftar kosong (kecuali platform admin).
//	@Tags			Auth
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200		{object}	respond.SuccessBody{data=[]dto.AppResponse}
//	@Failure		401,403	{object}	respond.ErrorBody
//	@Router			/api/v1/auth/apps [get]
func (h *AuthHandler) Apps(c *gin.Context) {
	actor, _ := middleware.PrincipalFrom(c)
	items, err := h.useCase.Apps(c.Request.Context(), actor)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "apps", items)
}
