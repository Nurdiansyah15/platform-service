package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/solutionclient"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/respond"
)

type SolutionClientHandler struct {
	useCase *solutionclient.SolutionClientUseCase
}

func NewSolutionClientHandler(useCase *solutionclient.SolutionClientUseCase) *SolutionClientHandler {
	return &SolutionClientHandler{useCase: useCase}
}

// Create godoc
//
//	@Summary		Create solution client
//	@Description	client_secret hanya ditampilkan sekali di respons ini.
//	@Tags			Solution Clients
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			solutionId		path		string								true	"Solution ID"
//	@Param			body			body		dto.CreateSolutionClientRequest		true	"Client"
//	@Success		201				{object}	respond.SuccessBody{data=dto.SolutionClientWithSecretResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/solutions/{solutionId}/clients [post]
func (h *SolutionClientHandler) Create(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.CreateSolutionClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Create(c.Request.Context(), param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "solution client created", resp)
}

// List godoc
//
//	@Summary	List solution clients
//	@Tags		Solution Clients
//	@Security	BearerAuth
//	@Produce	json
//	@Param		solutionId		path		string	true	"Solution ID"
//	@Param		status			query		string	false	"active | revoked"
//	@Param		search			query		string	false	"Cari di name / client_id"
//	@Param		page			query		int		false	"Halaman"
//	@Param		limit			query		int		false	"Jumlah per halaman (maks 100)"
//	@Param		sort_by			query		string	false	"name | created_at | last_used_at"
//	@Param		sort_type		query		string	false	"asc | desc"
//	@Success	200				{object}	respond.SuccessBody{data=[]dto.SolutionClientResponse,meta=dto.Meta}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/solutions/{solutionId}/clients [get]
func (h *SolutionClientHandler) List(c *gin.Context) {
	var param dto.SolutionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.ListSolutionClientQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	items, meta, err := h.useCase.List(c.Request.Context(), param.SolutionID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.SuccessWithMeta(c, http.StatusOK, "solution clients", items, meta)
}

// Get godoc
//
//	@Summary	Get solution client
//	@Tags		Solution Clients
//	@Security	BearerAuth
//	@Produce	json
//	@Param		clientId		path		string	true	"Solution client ID (UUID)"
//	@Success	200				{object}	respond.SuccessBody{data=dto.SolutionClientResponse}
//	@Failure	401,403,404,422	{object}	respond.ErrorBody
//	@Router		/api/v1/solution-clients/{clientId} [get]
func (h *SolutionClientHandler) Get(c *gin.Context) {
	var param dto.SolutionClientIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Get(c.Request.Context(), param.ClientID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "solution client", resp)
}

// Update godoc
//
//	@Summary		Update solution client
//	@Description	Ubah nama / scope. Scope baru langsung berlaku untuk token yang sudah terbit.
//	@Tags			Solution Clients
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			clientId			path		string							true	"Solution client ID (UUID)"
//	@Param			body				body		dto.UpdateSolutionClientRequest	true	"Client"
//	@Success		200					{object}	respond.SuccessBody{data=dto.SolutionClientResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/solution-clients/{clientId} [patch]
func (h *SolutionClientHandler) Update(c *gin.Context) {
	var param dto.SolutionClientIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	var req dto.UpdateSolutionClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Update(c.Request.Context(), param.ClientID, req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "solution client updated", resp)
}

// Revoke godoc
//
//	@Summary		Revoke solution client
//	@Description	Permanen; token yang sudah terbit langsung ditolak. Idempotent.
//	@Tags			Solution Clients
//	@Security		BearerAuth
//	@Produce		json
//	@Param			clientId		path		string	true	"Solution client ID (UUID)"
//	@Success		200				{object}	respond.SuccessBody{data=dto.SolutionClientResponse}
//	@Failure		401,403,404,422	{object}	respond.ErrorBody
//	@Router			/api/v1/solution-clients/{clientId}/revoke [post]
func (h *SolutionClientHandler) Revoke(c *gin.Context) {
	var param dto.SolutionClientIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Revoke(c.Request.Context(), param.ClientID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "solution client revoked", resp)
}

// RotateSecret godoc
//
//	@Summary		Rotate client secret
//	@Description	Secret lama langsung tidak berlaku; secret baru hanya ditampilkan sekali.
//	@Tags			Solution Clients
//	@Security		BearerAuth
//	@Produce		json
//	@Param			clientId			path		string	true	"Solution client ID (UUID)"
//	@Success		200					{object}	respond.SuccessBody{data=dto.SolutionClientWithSecretResponse}
//	@Failure		401,403,404,409,422	{object}	respond.ErrorBody
//	@Router			/api/v1/solution-clients/{clientId}/rotate-secret [post]
func (h *SolutionClientHandler) RotateSecret(c *gin.Context) {
	var param dto.SolutionClientIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.RotateSecret(c.Request.Context(), param.ClientID)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.OK(c, "client secret rotated", resp)
}

// Token godoc
//
//	@Summary		OAuth2 token (client credentials)
//	@Description	RFC 6749 §4.4. Kredensial via HTTP Basic atau field body. Respons & error format OAuth2 standar (tanpa envelope).
//	@Tags			Service Auth
//	@Accept			x-www-form-urlencoded
//	@Produce		json
//	@Param			grant_type		formData	string	true	"client_credentials"
//	@Param			client_id		formData	string	false	"Client ID (bila tidak memakai Basic)"
//	@Param			client_secret	formData	string	false	"Client secret (bila tidak memakai Basic)"
//	@Param			scope			formData	string	false	"Subset scope, dipisah spasi"
//	@Success		200				{object}	dto.OAuthTokenResponse
//	@Failure		400,401			{object}	dto.OAuthError
//	@Router			/api/v1/oauth/token [post]
func (h *SolutionClientHandler) Token(c *gin.Context) {
	// RFC 6749 §5.1: respons token tidak boleh di-cache.
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")

	var req dto.ClientCredentialsRequest
	if err := c.ShouldBind(&req); err != nil {
		writeOAuthError(c, dto.NewOAuthInvalidRequest("malformed request body"))
		return
	}
	if id, secret, ok := c.Request.BasicAuth(); ok {
		if req.ClientID != "" || req.ClientSecret != "" {
			// RFC 6749 §2.3: client tidak boleh memakai lebih dari satu metode autentikasi.
			writeOAuthError(c, dto.NewOAuthInvalidRequest("use either HTTP Basic or body credentials, not both"))
			return
		}
		req.ClientID, req.ClientSecret, req.UsedBasicAuth = id, secret, true
	}

	resp, err := h.useCase.IssueToken(c.Request.Context(), req)
	if err != nil {
		var oauthErr *dto.OAuthError
		if errors.As(err, &oauthErr) {
			writeOAuthError(c, oauthErr)
			return
		}
		httperror.Handle(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func writeOAuthError(c *gin.Context, e *dto.OAuthError) {
	if e.Status == http.StatusUnauthorized && e.BasicAuth {
		c.Header("WWW-Authenticate", `Basic realm="platform"`)
	}
	c.Set("request_err", e)
	c.Set("request_err_code", e.Code)
	c.AbortWithStatusJSON(e.Status, e)
}
