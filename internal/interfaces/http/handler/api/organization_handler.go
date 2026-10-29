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

// Update godoc
//
//	@Summary		Update company
//	@Description	Ubah name / slug (field kosong tidak diubah). status hanya boleh diubah platform admin.
//	@Tags			Companies
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			companyId			path		string						true	"Company ID"
//	@Param			body				body		dto.UpdateCompanyRequest	true	"Company"
//	@Success		200					{object}	respond.SuccessBody{data=dto.CompanyResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies/{companyId} [patch]
func (h *OrganizationHandler) Update(c *gin.Context) {
	var param dto.CompanyIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.UpdateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.companies.Update(c.Request.Context(), actor, param.CompanyID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "company updated", resp)
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

// GetMember godoc
//
//	@Summary	Get company member
//	@Tags		Companies
//	@Security	BearerAuth
//	@Produce	json
//	@Param		companyId		path		string	true	"Company ID"
//	@Param		userId			path		string	true	"User ID"
//	@Success	200				{object}	respond.SuccessBody{data=dto.CompanyMemberResponse}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/companies/{companyId}/members/{userId} [get]
func (h *OrganizationHandler) GetMember(c *gin.Context) {
	var param dto.CompanyMemberParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.members.Get(c.Request.Context(), actor, param.CompanyID, param.UserID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "company member", resp)
}

// UpdateMember godoc
//
//	@Summary		Update company member
//	@Description	Ubah status member. inactive = member dikeluarkan (role assignment di company ini dihapus).
//	@Tags			Companies
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			companyId		path		string							true	"Company ID"
//	@Param			userId			path		string							true	"User ID"
//	@Param			body			body		dto.UpdateCompanyMemberRequest	true	"Member"
//	@Success		200				{object}	respond.SuccessBody{data=dto.CompanyMemberResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies/{companyId}/members/{userId} [patch]
func (h *OrganizationHandler) UpdateMember(c *gin.Context) {
	var param dto.CompanyMemberParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.UpdateCompanyMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.members.Update(c.Request.Context(), actor, param.CompanyID, param.UserID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "company member updated", resp)
}

// RemoveMember godoc
//
//	@Summary		Remove company member
//	@Description	Soft: status inactive + hapus role assignment user di company ini. Idempotent.
//	@Tags			Companies
//	@Security		BearerAuth
//	@Produce		json
//	@Param			companyId		path		string	true	"Company ID"
//	@Param			userId			path		string	true	"User ID"
//	@Success		200				{object}	respond.SuccessBody
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies/{companyId}/members/{userId} [delete]
func (h *OrganizationHandler) RemoveMember(c *gin.Context) {
	var param dto.CompanyMemberParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	if err := h.members.Remove(c.Request.Context(), actor, param.CompanyID, param.UserID); err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "company member removed", nil)
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

// Summary godoc
//
//	@Summary	Company summary
//	@Description	Jumlah member per status, branch, custom role, dan role assignment (untuk dashboard).
//	@Tags		Companies
//	@Security	BearerAuth
//	@Produce	json
//	@Param		companyId		path		string	true	"Company ID"
//	@Success	200				{object}	respond.SuccessBody{data=dto.CompanySummaryResponse}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/companies/{companyId}/summary [get]
func (h *OrganizationHandler) Summary(c *gin.Context) {
	var param dto.CompanyIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.companies.Summary(c.Request.Context(), actor, param.CompanyID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "company summary", resp)
}

// MemberCandidates godoc
//
//	@Summary	Search member candidates
//	@Description	Cari user terdaftar (active) berdasarkan email / nama yang belum menjadi member active / invited. Minimal 3 karakter, maksimal 10 hasil.
//	@Tags		Companies
//	@Security	BearerAuth
//	@Produce	json
//	@Param		companyId		path		string	true	"Company ID"
//	@Param		search			query		string	true	"Email / nama (min. 3 karakter)"
//	@Success	200				{object}	respond.SuccessBody{data=[]dto.MemberCandidateResponse}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/companies/{companyId}/member-candidates [get]
func (h *OrganizationHandler) MemberCandidates(c *gin.Context) {
	var param dto.CompanyIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.MemberCandidateQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, err := h.members.Candidates(c.Request.Context(), actor, param.CompanyID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "member candidates", items)
}

// UpdateBranch godoc
//
//	@Summary		Update branch
//	@Description	Ubah nama / kode / status. Branch milik company yang tidak bisa diakses dilaporkan 404.
//	@Tags			Organization
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			branchId			path		string						true	"Branch ID"
//	@Param			body				body		dto.UpdateOrgUnitRequest	true	"Branch"
//	@Success		200					{object}	respond.SuccessBody{data=dto.BranchResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/branches/{branchId} [patch]
func (h *OrganizationHandler) UpdateBranch(c *gin.Context) {
	var param dto.BranchIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.UpdateOrgUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.branches.Update(c.Request.Context(), actor, param.BranchID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "branch updated", resp)
}

// DeleteBranch godoc
//
//	@Summary		Delete branch
//	@Description	Soft delete. Role assignment ber-scope branch ini ikut dihapus.
//	@Tags			Organization
//	@Security		BearerAuth
//	@Produce		json
//	@Param			branchId		path		string	true	"Branch ID"
//	@Success		200				{object}	respond.SuccessBody
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/branches/{branchId} [delete]
func (h *OrganizationHandler) DeleteBranch(c *gin.Context) {
	var param dto.BranchIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	if err := h.branches.Delete(c.Request.Context(), actor, param.BranchID); err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "branch deleted", nil)
}
