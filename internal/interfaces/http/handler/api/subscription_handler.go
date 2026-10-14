package api

import (
	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/subscription"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/middleware"
	"platform-service/internal/interfaces/http/respond"
)

type SubscriptionHandler struct {
	useCase *subscription.SubscriptionUseCase
}

func NewSubscriptionHandler(useCase *subscription.SubscriptionUseCase) *SubscriptionHandler {
	return &SubscriptionHandler{useCase: useCase}
}

// Create godoc
//
//	@Summary		Create subscription
//	@Description	Membekukan benefit plan ke snapshot. Maksimal satu subscription hidup (trialing/active/past_due) per tenant per solution.
//	@Tags			Subscriptions
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			tenantId			path		string							true	"Tenant ID"
//	@Param			body				body		dto.CreateSubscriptionRequest	true	"Subscription"
//	@Success		201					{object}	respond.SuccessBody{data=dto.SubscriptionResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/tenants/{tenantId}/subscriptions [post]
func (h *SubscriptionHandler) Create(c *gin.Context) {
	var param dto.TenantIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Create(c.Request.Context(), param.TenantID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "subscription created", resp)
}

// List godoc
//
//	@Summary		List tenant subscriptions
//	@Description	Token service hanya melihat subscription solution miliknya.
//	@Tags			Subscriptions
//	@Security		BearerAuth
//	@Produce		json
//	@Param			tenantId		path		string	true	"Tenant ID"
//	@Param			status			query		string	false	"trialing | active | past_due | canceled | expired"
//	@Param			page			query		int		false	"Halaman"
//	@Param			limit			query		int		false	"Jumlah per halaman (maks 100)"
//	@Param			sort_by			query		string	false	"created_at | status | current_period_end"
//	@Param			sort_type		query		string	false	"asc | desc"
//	@Success		200				{object}	respond.SuccessBody{data=[]dto.SubscriptionResponse,meta=dto.Meta}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/tenants/{tenantId}/subscriptions [get]
func (h *SubscriptionHandler) List(c *gin.Context) {
	var param dto.TenantIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ListSubscriptionQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, meta, err := h.useCase.List(c.Request.Context(), actor, param.TenantID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.SuccessWithMeta(c, 200, "subscriptions", items, meta)
}

// ChangePlan godoc
//
//	@Summary		Change subscription plan
//	@Description	Mengganti snapshot benefit plan. Periode baru dimulai saat ini.
//	@Tags			Subscriptions
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			subscriptionId		path		string					true	"Subscription ID"
//	@Param			body				body		dto.ChangePlanRequest	true	"Plan baru"
//	@Success		200					{object}	respond.SuccessBody{data=dto.SubscriptionResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/subscriptions/{subscriptionId}/change-plan [post]
func (h *SubscriptionHandler) ChangePlan(c *gin.Context) {
	var param dto.SubscriptionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ChangePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.ChangePlan(c.Request.Context(), param.SubscriptionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "subscription plan changed", resp)
}

// Cancel godoc
//
//	@Summary	Cancel subscription
//	@Tags		Subscriptions
//	@Security	BearerAuth
//	@Produce	json
//	@Param		subscriptionId		path		string	true	"Subscription ID"
//	@Success	200					{object}	respond.SuccessBody{data=dto.SubscriptionResponse}
//	@Failure	401,403,404,409,422	{object}	respond.ErrorBody
//	@Router		/api/v1/subscriptions/{subscriptionId}/cancel [post]
func (h *SubscriptionHandler) Cancel(c *gin.Context) {
	var param dto.SubscriptionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Cancel(c.Request.Context(), param.SubscriptionID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "subscription canceled", resp)
}

// ListAddons godoc
//
//	@Summary	List subscription add-ons
//	@Tags		Subscriptions
//	@Security	BearerAuth
//	@Produce	json
//	@Param		subscriptionId	path		string	true	"Subscription ID"
//	@Success	200				{object}	respond.SuccessBody{data=[]dto.SubscriptionAddonResponse}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/subscriptions/{subscriptionId}/addons [get]
func (h *SubscriptionHandler) ListAddons(c *gin.Context) {
	var param dto.SubscriptionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, err := h.useCase.ListAddons(c.Request.Context(), actor, param.SubscriptionID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "subscription add-ons", items)
}

// AddAddon godoc
//
//	@Summary		Add add-on to subscription
//	@Description	Membekukan benefit add-on ke snapshot. Add-on yang pernah dibatalkan diaktifkan ulang.
//	@Tags			Subscriptions
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			subscriptionId		path		string								true	"Subscription ID"
//	@Param			body				body		dto.AddSubscriptionAddonRequest		true	"Add-on"
//	@Success		201					{object}	respond.SuccessBody{data=dto.SubscriptionAddonResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/subscriptions/{subscriptionId}/addons [post]
func (h *SubscriptionHandler) AddAddon(c *gin.Context) {
	var param dto.SubscriptionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.AddSubscriptionAddonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.AddAddon(c.Request.Context(), param.SubscriptionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "subscription add-on added", resp)
}
