package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"platform-service/internal/app/usecase/signingkey"
	"platform-service/internal/interfaces/http/httperror"
)

// jwksCacheControl — solution boleh meng-cache JWKS; key baru tetap terambil
// dalam 5 menit, jauh di bawah masa transisi key retiring.
const jwksCacheControl = "public, max-age=300"

type JWKSHandler struct {
	useCase *signingkey.SigningKeyUseCase
}

func NewJWKSHandler(useCase *signingkey.SigningKeyUseCase) *JWKSHandler {
	return &JWKSHandler{useCase: useCase}
}

// JWKS godoc
//
//	@Summary		JSON Web Key Set
//	@Description	Public key untuk memverifikasi token platform (key active & retiring). Respons tanpa envelope (RFC 7517).
//	@Tags			Service Auth
//	@Produce		json
//	@Success		200	{object}	dto.JWKSResponse
//	@Router			/api/v1/.well-known/jwks.json [get]
func (h *JWKSHandler) JWKS(c *gin.Context) {
	resp, err := h.useCase.JWKS(c.Request.Context())
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	c.Header("Cache-Control", jwksCacheControl)
	c.JSON(http.StatusOK, resp)
}
