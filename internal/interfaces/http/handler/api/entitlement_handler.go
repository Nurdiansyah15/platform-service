package api

import (
	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/entitlement"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/middleware"
	"platform-service/internal/interfaces/http/respond"
)

type EntitlementHandler struct {
	useCase *entitlement.EntitlementUseCase
}

func NewEntitlementHandler(useCase *entitlement.EntitlementUseCase) *EntitlementHandler {
	return &EntitlementHandler{useCase: useCase}
}

// List godoc
//
//	@Summary		Effective entitlements
//	@Description	Dihitung dari snapshot subscription (sum: plan + add-on × quantity; max/min; any_true; override: add-on terbaru). -1 = unlimited. Benefit di-meter memuat limit, used, remaining.
//	@Tags			Entitlements
//	@Security		BearerAuth
//	@Produce		json
//	@Param			tenantId		path		string	true	"Tenant ID"
//	@Param			solutionId		path		string	true	"Solution ID"
//	@Success		200				{object}	respond.SuccessBody{data=dto.EntitlementsResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/tenants/{tenantId}/solutions/{solutionId}/entitlements [get]
func (h *EntitlementHandler) List(c *gin.Context) {
	var param dto.TenantSolutionParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.List(c.Request.Context(), actor, param.TenantID, param.SolutionID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "entitlements", resp)
}

// Get godoc
//
//	@Summary	Effective entitlement of one benefit
//	@Tags		Entitlements
//	@Security	BearerAuth
//	@Produce	json
//	@Param		tenantId		path		string	true	"Tenant ID"
//	@Param		solutionId		path		string	true	"Solution ID"
//	@Param		benefitCode		path		string	true	"Benefit code"
//	@Success	200				{object}	respond.SuccessBody{data=dto.EntitlementResponse}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/tenants/{tenantId}/solutions/{solutionId}/entitlements/{benefitCode} [get]
func (h *EntitlementHandler) Get(c *gin.Context) {
	var param dto.EntitlementBenefitParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	resp, err := h.useCase.Get(c.Request.Context(), actor, param.TenantID, param.SolutionID, param.BenefitCode)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "entitlement", resp)
}
