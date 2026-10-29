package api

import (
	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/authz"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/middleware"
	"platform-service/internal/interfaces/http/respond"
)

type AuthzHandler struct {
	useCase *authz.AuthzUseCase
}

func NewAuthzHandler(useCase *authz.AuthzUseCase) *AuthzHandler {
	return &AuthzHandler{useCase: useCase}
}

// AssignRole godoc
//
//	@Summary		Assign role to company member
//	@Description	User wajib member active; company wajib subscribe solution role; custom role harus milik company; scope_id harus milik company (company / company / branch).
//	@Tags			Authorization
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			companyId			path		string							true	"Company ID"
//	@Param			userId				path		string							true	"User ID"
//	@Param			body				body		dto.CreateRoleAssignmentRequest	true	"Assignment"
//	@Success		201					{object}	respond.SuccessBody{data=dto.RoleAssignmentResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies/{companyId}/users/{userId}/role-assignments [post]
func (h *AuthzHandler) AssignRole(c *gin.Context) {
	var param dto.CompanyUserParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.CreateRoleAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.AssignRole(c.Request.Context(), actor, param.CompanyID, param.UserID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "role assigned", resp)
}

// Check godoc
//
//	@Summary		Check permission
//	@Description	Token service: wajib company_id & user_id (solution = milik client). Token user: mengecek diri sendiri pada company aktif (cid). Scope default company; assignment company berlaku untuk branch di bawahnya.
//	@Tags			Authorization
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body			body		dto.AuthzCheckRequest	true	"Permission check"
//	@Success		200				{object}	respond.SuccessBody{data=dto.AuthzCheckResult}
//	@Failure		401,403,422		{object}	respond.ErrorBody
//	@Router			/api/v1/authz/check [post]
func (h *AuthzHandler) Check(c *gin.Context) {
	var req dto.AuthzCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.Check(c.Request.Context(), actor, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "authz check", resp)
}

// CheckBatch godoc
//
//	@Summary	Check permissions in batch
//	@Tags		Authorization
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		body		body		dto.AuthzCheckBatchRequest	true	"Maks 100 cek"
//	@Success	200			{object}	respond.SuccessBody{data=dto.AuthzCheckBatchResponse}
//	@Failure	401,403,422	{object}	respond.ErrorBody
//	@Router		/api/v1/authz/check-batch [post]
func (h *AuthzHandler) CheckBatch(c *gin.Context) {
	var req dto.AuthzCheckBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.CheckBatch(c.Request.Context(), actor, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "authz check batch", resp)
}

// ListAssignments godoc
//
//	@Summary		List role assignments of a company member
//	@Description	Token service hanya melihat role milik solution-nya; token user boleh memfilter solution_id.
//	@Tags			Authorization
//	@Security		BearerAuth
//	@Produce		json
//	@Param			companyId		path		string	true	"Company ID"
//	@Param			userId			path		string	true	"User ID"
//	@Param			solution_id		query		string	false	"Filter solution"
//	@Success		200				{object}	respond.SuccessBody{data=[]dto.RoleAssignmentDetailResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies/{companyId}/users/{userId}/role-assignments [get]
func (h *AuthzHandler) ListAssignments(c *gin.Context) {
	var param dto.CompanyUserParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.SolutionFilterQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, err := h.useCase.ListAssignments(c.Request.Context(), actor, param.CompanyID, param.UserID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "role assignments", items)
}

// RevokeAssignment godoc
//
//	@Summary		Revoke role assignment
//	@Description	Assignment milik company yang tidak bisa diakses (atau role solution lain untuk token service) dilaporkan 404.
//	@Tags			Authorization
//	@Security		BearerAuth
//	@Produce		json
//	@Param			assignmentId	path		string	true	"Role assignment ID"
//	@Success		200				{object}	respond.SuccessBody
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/role-assignments/{assignmentId} [delete]
func (h *AuthzHandler) RevokeAssignment(c *gin.Context) {
	var param dto.RoleAssignmentIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	if err := h.useCase.RevokeAssignment(c.Request.Context(), actor, param.AssignmentID); err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "role assignment revoked", nil)
}

// EffectivePermissions godoc
//
//	@Summary		Effective permissions of a company member
//	@Description	Seluruh permission user di company untuk satu solution beserta role & scope asalnya. Token user wajib solution_id; token service memakai solution client.
//	@Tags			Authorization
//	@Security		BearerAuth
//	@Produce		json
//	@Param			companyId		path		string	true	"Company ID"
//	@Param			userId			path		string	true	"User ID"
//	@Param			solution_id		query		string	false	"Solution (wajib untuk token user)"
//	@Success		200				{object}	respond.SuccessBody{data=[]dto.EffectivePermissionResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/companies/{companyId}/users/{userId}/effective-permissions [get]
func (h *AuthzHandler) EffectivePermissions(c *gin.Context) {
	var param dto.CompanyUserParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.SolutionFilterQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, err := h.useCase.EffectivePermissions(c.Request.Context(), actor, param.CompanyID, param.UserID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "effective permissions", items)
}
