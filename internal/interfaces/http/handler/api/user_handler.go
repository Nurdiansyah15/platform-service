package api

import (
	"github.com/gin-gonic/gin"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/user"
	"platform-service/internal/interfaces/http/httperror"
	"platform-service/internal/interfaces/http/respond"
)

type UserHandler struct {
	useCase *user.UserUseCase
}

func NewUserHandler(useCase *user.UserUseCase) *UserHandler {
	return &UserHandler{useCase: useCase}
}

// Create godoc
//
//	@Summary	Create user
//	@Tags		Users
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		body			body		dto.CreateUserRequest	true	"User"
//	@Success	201				{object}	respond.SuccessBody{data=dto.UserResponse}
//	@Failure	401,403,409,422	{object}	respond.ErrorBody
//	@Router		/api/v1/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperror.Handle(c, err)
		return
	}
	resp, err := h.useCase.Create(c.Request.Context(), req)
	if err != nil {
		httperror.Handle(c, err)
		return
	}
	respond.Created(c, "user created", resp)
}
