package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/zayi-group/go-spirnt1/internal/app"
	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

type createPostRequest struct {
	AuthorID string `json:"authorID"`
	Text     string `json:"text"`
}

type likePostRequest struct {
	UserID string `json:"userID"`
}

type postResponse struct {
	ID         string    `json:"id"`
	AuthorID   string    `json:"authorID"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"createdAt"`
	LikesCount int       `json:"likesCount"`
}

type PostHandler struct {
	service *app.PostService
}

func NewPostHandler(service *app.PostService) *PostHandler {
	return &PostHandler{
		service: service,
	}
}

func (h *PostHandler) CreatePost(ctx *gin.Context) {

	var request createPostRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON body",
		})
		return
	}

	post, err := h.service.CreatePost(request.AuthorID, request.Text)
	if err != nil {
		if errors.Is(err, app.ErrInvalidPost) {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid post data",
			})
			return
		}
		if errors.Is(err, domain.ErrUserNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "author not found",
			})
			return
		}
		if errors.Is(err, domain.ErrPostAlreadyExists) {
			ctx.JSON(http.StatusConflict, gin.H{
				"error": "post with this ID already exists",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create the post",
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"id":        post.ID,
		"authorID":  post.AuthorID,
		"text":      post.Text,
		"createdAt": post.CreatedAt,
	})

}

func (h *PostHandler) List(ctx *gin.Context) {

	responses := make([]postResponse, 0)

	posts, err := h.service.List()

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list posts",
		})
		return
	}

	for _, post := range posts {
		responses = append(responses, postResponse{
			ID:         post.ID,
			AuthorID:   post.AuthorID,
			Text:       post.Text,
			CreatedAt:  post.CreatedAt,
			LikesCount: len(post.Likes),
		})
	}

	ctx.JSON(http.StatusOK, responses)
}

func (h *PostHandler) GetPost(ctx *gin.Context) {
	id := ctx.Param("id")

	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid post id",
		})
		return
	}

	post, err := h.service.GetPost(id)

	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "post not found",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get the post",
		})
		return
	}

	response := postResponse{
		ID:         post.ID,
		AuthorID:   post.AuthorID,
		Text:       post.Text,
		CreatedAt:  post.CreatedAt,
		LikesCount: len(post.Likes),
	}

	ctx.JSON(http.StatusOK, response)
}

func (h *PostHandler) LikePost(ctx *gin.Context) {
	postID := ctx.Param("id")

	var request likePostRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON body",
		})
		return
	}

	err := h.service.LikePost(request.UserID, postID)
	if err != nil {
		if errors.Is(err, app.ErrInvalidLike) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid like data"})
			return
		}

		if errors.Is(err, domain.ErrUserNotFound) ||
			errors.Is(err, domain.ErrPostNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "user or post not found"})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to like the post"})
		return
	}

	if postID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid post id",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
