package api

import (
	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/tenant"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/middleware"
	"platform-service/internal/interfaces/http/respond"
)

type TenantHandler struct {
	tenants *tenant.TenantUseCase
	members *tenant.MemberUseCase
}

func NewTenantHandler(tenants *tenant.TenantUseCase, members *tenant.MemberUseCase) *TenantHandler {
	return &TenantHandler{tenants: tenants, members: members}
}

// Create godoc
//
//	@Summary		Create tenant
//	@Description	Pembuat (non platform admin) otomatis menjadi member active.
//	@Tags			Tenants
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body			body		dto.CreateTenantRequest	true	"Tenant"
//	@Success		201				{object}	respond.SuccessBody{data=dto.TenantResponse}
//	@Failure		401,403,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/tenants [post]
func (h *TenantHandler) Create(c *gin.Context) {
	var req dto.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.tenants.Create(c.Request.Context(), actor, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "tenant created", resp)
}

// List godoc
//
//	@Summary		List tenants
//	@Description	Platform admin melihat semua tenant; user lain hanya tenant tempat ia member active.
//	@Tags			Tenants
//	@Security		BearerAuth
//	@Produce		json
//	@Param			status		query		string	false	"active | suspended | closed"
//	@Param			search		query		string	false	"Cari di name / slug"
//	@Param			page		query		int		false	"Halaman"
//	@Param			limit		query		int		false	"Jumlah per halaman (maks 100)"
//	@Param			sort_by		query		string	false	"name | slug | created_at"
//	@Param			sort_type	query		string	false	"asc | desc"
//	@Success		200			{object}	respond.SuccessBody{data=[]dto.TenantResponse,meta=dto.Meta}
//	@Failure		401,403,422	{object}	respond.ErrorBody
//	@Router			/api/v1/tenants [get]
func (h *TenantHandler) List(c *gin.Context) {
	var req dto.ListTenantQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, meta, err := h.tenants.List(c.Request.Context(), actor, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.SuccessWithMeta(c, 200, "tenants", items, meta)
}

// Get godoc
//
//	@Summary		Get tenant
//	@Description	Platform admin, atau member active yang token-nya ber-tid tenant ini.
//	@Tags			Tenants
//	@Security		BearerAuth
//	@Produce		json
//	@Param			tenantId		path		string	true	"Tenant ID"
//	@Success		200				{object}	respond.SuccessBody{data=dto.TenantResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/tenants/{tenantId} [get]
func (h *TenantHandler) Get(c *gin.Context) {
	var param dto.TenantIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.tenants.Get(c.Request.Context(), actor, param.TenantID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "tenant", resp)
}

// ListMembers godoc
//
//	@Summary	List tenant members
//	@Tags		Tenants
//	@Security	BearerAuth
//	@Produce	json
//	@Param		tenantId		path		string	true	"Tenant ID"
//	@Param		status			query		string	false	"invited | active | inactive"
//	@Param		search			query		string	false	"Cari di email / nama user"
//	@Param		page			query		int		false	"Halaman"
//	@Param		limit			query		int		false	"Jumlah per halaman (maks 100)"
//	@Param		sort_by			query		string	false	"email | name | joined_at"
//	@Param		sort_type		query		string	false	"asc | desc"
//	@Success	200				{object}	respond.SuccessBody{data=[]dto.TenantMemberResponse,meta=dto.Meta}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/tenants/{tenantId}/members [get]
func (h *TenantHandler) ListMembers(c *gin.Context) {
	var param dto.TenantIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ListTenantMemberQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, meta, err := h.members.List(c.Request.Context(), actor, param.TenantID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.SuccessWithMeta(c, 200, "tenant members", items, meta)
}

// AddMember godoc
//
//	@Summary		Add tenant member
//	@Description	Isi salah satu user_id atau email (user harus sudah terdaftar). Member inactive diaktifkan kembali.
//	@Tags			Tenants
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			tenantId			path		string						true	"Tenant ID"
//	@Param			body				body		dto.AddTenantMemberRequest	true	"Member"
//	@Success		201					{object}	respond.SuccessBody{data=dto.TenantMemberResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/tenants/{tenantId}/members [post]
func (h *TenantHandler) AddMember(c *gin.Context) {
	var param dto.TenantIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.AddTenantMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.members.Add(c.Request.Context(), actor, param.TenantID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "tenant member added", resp)
}
