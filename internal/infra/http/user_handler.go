package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/zayi-group/go-spirnt1/internal/app"
	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

type registerUserRequest struct {
	Username string `json:"username"`
}

type UserHandler struct {
	service *app.UserService
}

func NewUserHandler(service *app.UserService) *UserHandler {
	return &UserHandler{
		service: service,
	}
}

func (h *UserHandler) RegisterUser(ctx *gin.Context) {
	var request registerUserRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}

	user, err := h.service.RegisterUser(request.Username)

	if err != nil {
		if errors.Is(err, app.ErrInvalidRegistration) {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid username",
			})
			return
		}
		if errors.Is(err, domain.ErrUsernameTaken) {
			ctx.JSON(http.StatusConflict, gin.H{
				"error": "username already taken",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to register the user",
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"username": user.Username,
	})

}

func (h *UserHandler) List(ctx *gin.Context) {

	users, err := h.service.List()

	if err != nil {

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})

		return
	}

	ctx.JSON(http.StatusOK, users)
}
