package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter(userHandler *UserHandler, postHandler *PostHandler) *gin.Engine {
	router := gin.Default()

	router.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	v1 := router.Group("/api/v1")
	{
		//Users
		v1.POST("/register", userHandler.RegisterUser)
		v1.GET("/users", userHandler.List)

		//Post
		v1.POST("/posts", postHandler.CreatePost)
		v1.GET("/posts", postHandler.List)
		v1.GET("/posts/:id", postHandler.GetPost)

		v1.POST("/posts/:id/like", postHandler.LikePost)
	}

	return router
}
