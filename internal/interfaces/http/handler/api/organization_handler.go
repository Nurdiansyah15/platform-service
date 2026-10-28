package api

import (
	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/organization"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/middleware"
	"platform-service/internal/interfaces/http/respond"
)

type OrganizationHandler struct {
	companies *organization.CompanyUseCase
	members   *organization.CompanyMemberUseCase
	branches  *organization.BranchUseCase
}

func NewOrganizationHandler(companies *organization.CompanyUseCase, members *organization.CompanyMemberUseCase, branches *organization.BranchUseCase) *OrganizationHandler {
	return &OrganizationHandler{companies: companies, members: members, branches: branches}
}

// Create godoc
//
//	@Summary		Create company
//	@Description	Pembuat (non platform admin) otomatis menjadi member active.
//	@Tags			Companies
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body			body		dto.CreateCompanyRequest	true	"Company"
//	@Success		201				{object}	respond.SuccessBody{data=dto.CompanyResponse}
//	@Failure		401,403,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies [post]
func (h *OrganizationHandler) Create(c *gin.Context) {
	var req dto.CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.companies.Create(c.Request.Context(), actor, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "company created", resp)
}

// List godoc
//
//	@Summary		List companies
//	@Description	Platform admin melihat semua company; user lain hanya company tempat ia member active.
//	@Tags			Companies
//	@Security		BearerAuth
//	@Produce		json
//	@Param			status		query		string	false	"active | suspended | closed"
//	@Param			search		query		string	false	"Cari di name / slug"
//	@Param			page		query		int		false	"Halaman"
//	@Param			limit		query		int		false	"Jumlah per halaman (maks 100)"
//	@Param			sort_by		query		string	false	"name | slug | created_at"
//	@Param			sort_type	query		string	false	"asc | desc"
//	@Success		200			{object}	respond.SuccessBody{data=[]dto.CompanyResponse,meta=dto.Meta}
//	@Failure		401,403,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies [get]
func (h *OrganizationHandler) List(c *gin.Context) {
	var req dto.ListCompanyQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, meta, err := h.companies.List(c.Request.Context(), actor, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.SuccessWithMeta(c, 200, "companies", items, meta)
}

// Get godoc
//
//	@Summary		Get company
//	@Description	Platform admin, atau member active yang token-nya ber-cid company ini.
//	@Tags			Companies
//	@Security		BearerAuth
//	@Produce		json
//	@Param			companyId		path		string	true	"Company ID"
//	@Success		200				{object}	respond.SuccessBody{data=dto.CompanyResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies/{companyId} [get]
func (h *OrganizationHandler) Get(c *gin.Context) {
	var param dto.CompanyIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.companies.Get(c.Request.Context(), actor, param.CompanyID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "company", resp)
}

// ListMembers godoc
//
//	@Summary	List company members
//	@Tags		Companies
//	@Security	BearerAuth
//	@Produce	json
//	@Param		companyId		path		string	true	"Company ID"
//	@Param		status			query		string	false	"invited | active | inactive"
//	@Param		search			query		string	false	"Cari di email / nama user"
//	@Param		page			query		int		false	"Halaman"
//	@Param		limit			query		int		false	"Jumlah per halaman (maks 100)"
//	@Param		sort_by			query		string	false	"email | name | joined_at"
//	@Param		sort_type		query		string	false	"asc | desc"
//	@Success	200				{object}	respond.SuccessBody{data=[]dto.CompanyMemberResponse,meta=dto.Meta}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/companies/{companyId}/members [get]
func (h *OrganizationHandler) ListMembers(c *gin.Context) {
	var param dto.CompanyIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ListCompanyMemberQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, meta, err := h.members.List(c.Request.Context(), actor, param.CompanyID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.SuccessWithMeta(c, 200, "company members", items, meta)
}

// AddMember godoc
//
//	@Summary		Add company member
//	@Description	Isi salah satu user_id atau email (user harus sudah terdaftar). Member inactive diaktifkan kembali.
//	@Tags			Companies
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			companyId			path		string						true	"Company ID"
//	@Param			body				body		dto.AddCompanyMemberRequest	true	"Member"
//	@Success		201					{object}	respond.SuccessBody{data=dto.CompanyMemberResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies/{companyId}/members [post]
func (h *OrganizationHandler) AddMember(c *gin.Context) {
	var param dto.CompanyIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.AddCompanyMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.members.Add(c.Request.Context(), actor, param.CompanyID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "company member added", resp)
}

// CreateBranch godoc
//
//	@Summary		Create branch
//	@Description	code dinormalisasi UPPERCASE, unik per company.
//	@Tags			Organization
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			companyId			path		string						true	"Company ID"
//	@Param			body				body		dto.CreateOrgUnitRequest	true	"Branch"
//	@Success		201					{object}	respond.SuccessBody{data=dto.BranchResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies/{companyId}/branches [post]
func (h *OrganizationHandler) CreateBranch(c *gin.Context) {
	var param dto.CompanyIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.CreateOrgUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.branches.Create(c.Request.Context(), actor, param.CompanyID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "branch created", resp)
}

// ListBranches godoc
//
//	@Summary	List branches
//	@Tags		Organization
//	@Security	BearerAuth
//	@Produce	json
//	@Param		companyId		path		string	true	"Company ID"
//	@Param		status			query		string	false	"active | inactive"
//	@Param		search			query		string	false	"Cari di code / name"
//	@Param		page			query		int		false	"Halaman"
//	@Param		limit			query		int		false	"Jumlah per halaman (maks 100)"
//	@Param		sort_by			query		string	false	"code | name | created_at"
//	@Param		sort_type		query		string	false	"asc | desc"
//	@Success	200				{object}	respond.SuccessBody{data=[]dto.BranchResponse,meta=dto.Meta}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/companies/{companyId}/branches [get]
func (h *OrganizationHandler) ListBranches(c *gin.Context) {
	var param dto.CompanyIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ListOrgUnitQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, meta, err := h.branches.List(c.Request.Context(), actor, param.CompanyID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.SuccessWithMeta(c, 200, "branches", items, meta)
}
