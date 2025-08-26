package routes

import (
	"github.com/Sayan80bayev/go-project/pkg/middleware"
	"github.com/gin-gonic/gin"
	"postService/internal/bootstrap"
	"postService/internal/delivery"
	"postService/internal/service"
)

func SetupPostRoutes(r *gin.Engine, c *bootstrap.Container) {
	cfg := c.Config
	minio := c.FileStorage
	redis := c.Redis
	producer := c.Producer
	postRepo := c.PostRepository

	postService := service.NewPostService(postRepo, minio, redis, producer)
	postHandler := delivery.NewPostHandler(postService, cfg)

	r.GET("api/v1/posts", postHandler.GetPosts)
	r.GET("api/v1/posts/:id", postHandler.GetPostByID)

	postRoutes := r.Group("api/v1/posts", middleware.AuthMiddleware(c.JWKSurl))
	{
		postRoutes.POST("/", postHandler.CreatePost)
		postRoutes.PUT("/:id", postHandler.UpdatePost)
		postRoutes.DELETE("/:id", postHandler.DeletePost)
	}
}
