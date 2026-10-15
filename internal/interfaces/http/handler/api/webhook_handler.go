package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/webhook"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/middleware"
	"platform-service/internal/interfaces/http/respond"
)

type WebhookHandler struct {
	useCase *webhook.WebhookUseCase
}

func NewWebhookHandler(useCase *webhook.WebhookUseCase) *WebhookHandler {
	return &WebhookHandler{useCase: useCase}
}

// Create godoc
//
//	@Summary		Create webhook endpoint
//	@Description	Secret HMAC hanya ditampilkan sekali di respons ini. Penerima memverifikasi X-Platform-Signature = "sha256=" + hex(HMAC-SHA256(secret, timestamp + "." + body)).
//	@Tags			Webhooks
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			solutionId		path		string						true	"Solution ID"
//	@Param			body			body		dto.CreateWebhookRequest	true	"Webhook"
//	@Success		201				{object}	respond.SuccessBody{data=dto.WebhookWithSecretResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/solutions/{solutionId}/webhooks [post]
func (h *WebhookHandler) Create(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Create(c.Request.Context(), param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "webhook created", resp)
}

// ListEvents godoc
//
//	@Summary		List events (polling / replay)
//	@Description	Event yang relevan untuk solution pemanggil, urut occurred_at naik. Lanjutkan dengan `cursor` = meta.next_cursor.
//	@Tags			Events
//	@Security		BearerAuth
//	@Produce		json
//	@Param			since		query		string	false	"RFC3339; event dengan occurred_at >= since"
//	@Param			event_type	query		string	false	"Filter tipe event"
//	@Param			tenant_id	query		string	false	"Filter tenant"
//	@Param			cursor		query		string	false	"meta.next_cursor dari halaman sebelumnya"
//	@Param			limit		query		int		false	"1–200, default 50"
//	@Success		200			{object}	respond.SuccessBody{data=[]dto.EventItem,meta=dto.CursorMeta}
//	@Failure		401,403,422	{object}	respond.ErrorBody
//	@Router			/api/v1/events [get]
func (h *WebhookHandler) ListEvents(c *gin.Context) {
	var q dto.ListEventsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		httperror.Handle(c, err)
		return
	}
	actor, _ := middleware.PrincipalFrom(c)
	items, meta, err := h.useCase.ListEvents(c.Request.Context(), actor, q)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.SuccessWithMeta(c, http.StatusOK, "events retrieved", items, meta)
}
