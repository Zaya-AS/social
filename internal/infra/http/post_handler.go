package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gitlab.com/zayi-group/go-spirnt1/internal/app"
	"gitlab.com/zayi-group/go-spirnt1/internal/app/likes"
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
	service     *app.PostService
	likeService *likes.Service
}

func NewPostHandler(service *app.PostService, likeService *likes.Service) *PostHandler {
	return &PostHandler{
		service:     service,
		likeService: likeService,
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
	if postID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid post id",
		})
		return
	}

	var request likePostRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON body",
		})
		return
	}

	cmd := domain.LikeCommand{
		CommandID: uuid.NewString(),
		PostID:    postID,
		UserID:    request.UserID,
		CreatedAt: time.Now(),
	}

	result, err := h.likeService.EnqueueLike(ctx.Request.Context(), cmd)
	if err != nil {
		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			return
		}

		if errors.Is(err, likes.ErrQueueFull) ||
			errors.Is(err, likes.ErrQueueClosed) {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "like processing is unavailable",
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to submit like",
		})
		return
	}

	if result.Err != nil {
		if errors.Is(result.Err, likes.ErrInvalidLike) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid like data"})
			return
		}

		if errors.Is(result.Err, domain.ErrUserNotFound) ||
			errors.Is(result.Err, domain.ErrPostNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "user or post not found"})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to like the post",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}
