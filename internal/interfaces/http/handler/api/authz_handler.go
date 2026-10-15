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
//	@Summary		Assign role to tenant member
//	@Description	User wajib member active; tenant wajib subscribe solution role; custom role harus milik tenant; scope_id harus milik tenant (tenant / company / branch).
//	@Tags			Authorization
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			tenantId			path		string							true	"Tenant ID"
//	@Param			userId				path		string							true	"User ID"
//	@Param			body				body		dto.CreateRoleAssignmentRequest	true	"Assignment"
//	@Success		201					{object}	respond.SuccessBody{data=dto.RoleAssignmentResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/tenants/{tenantId}/users/{userId}/role-assignments [post]
func (h *AuthzHandler) AssignRole(c *gin.Context) {
	var param dto.TenantUserParam
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
	resp, err := h.useCase.AssignRole(c.Request.Context(), actor, param.TenantID, param.UserID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "role assigned", resp)
}

// Check godoc
//
//	@Summary		Check permission
//	@Description	Token service: wajib tenant_id & user_id (solution = milik client). Token user: mengecek diri sendiri pada tenant aktif (tid). Scope default tenant; assignment company berlaku untuk branch di bawahnya.
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
