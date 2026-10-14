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
	branches  *organization.BranchUseCase
}

func NewOrganizationHandler(companies *organization.CompanyUseCase, branches *organization.BranchUseCase) *OrganizationHandler {
	return &OrganizationHandler{companies: companies, branches: branches}
}

// CreateCompany godoc
//
//	@Summary		Create company
//	@Description	code dinormalisasi UPPERCASE, unik per tenant.
//	@Tags			Organization
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			tenantId			path		string						true	"Tenant ID"
//	@Param			body				body		dto.CreateOrgUnitRequest	true	"Company"
//	@Success		201					{object}	respond.SuccessBody{data=dto.CompanyResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/tenants/{tenantId}/companies [post]
func (h *OrganizationHandler) CreateCompany(c *gin.Context) {
	var param dto.TenantIDParam
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
	resp, err := h.companies.Create(c.Request.Context(), actor, param.TenantID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "company created", resp)
}

// ListCompanies godoc
//
//	@Summary	List companies
//	@Tags		Organization
//	@Security	BearerAuth
//	@Produce	json
//	@Param		tenantId		path		string	true	"Tenant ID"
//	@Param		status			query		string	false	"active | inactive"
//	@Param		search			query		string	false	"Cari di code / name"
//	@Param		page			query		int		false	"Halaman"
//	@Param		limit			query		int		false	"Jumlah per halaman (maks 100)"
//	@Param		sort_by			query		string	false	"code | name | created_at"
//	@Param		sort_type		query		string	false	"asc | desc"
//	@Success	200				{object}	respond.SuccessBody{data=[]dto.CompanyResponse,meta=dto.Meta}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/tenants/{tenantId}/companies [get]
func (h *OrganizationHandler) ListCompanies(c *gin.Context) {
	var param dto.TenantIDParam
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
	items, meta, err := h.companies.List(c.Request.Context(), actor, param.TenantID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.SuccessWithMeta(c, 200, "companies", items, meta)
}

// CreateBranch godoc
//
//	@Summary		Create branch
//	@Description	code dinormalisasi UPPERCASE, unik per company. Company tenant lain → 404.
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
