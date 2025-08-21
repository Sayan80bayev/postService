package routes

import (
	"github.com/Sayan80bayev/go-project/pkg/middleware"
	"github.com/gin-gonic/gin"
	"postService/internal/bootstrap"
	"postService/internal/delivery"
	"postService/internal/repository"
	"postService/internal/service"
)

func SetupPostRoutes(r *gin.Engine, bs *bootstrap.Container) {
	cfg := bs.Config
	minio := bs.Minio
	redis := bs.Redis
	producer := bs.Producer

	postRepo := repository.GetPostRepository(bs.DB)

	postService := service.NewPostService(postRepo, minio, redis, producer)
	postHandler := delivery.NewPostHandler(postService, cfg)

	r.GET("api/v1/posts", postHandler.GetPosts)
	r.GET("api/v1/posts/:id", postHandler.GetPostByID)

	postRoutes := r.Group("api/v1/posts", middleware.AuthMiddleware(bs.JWKSURL))
	{
		postRoutes.POST("/", postHandler.CreatePost)
		postRoutes.PUT("/:id", postHandler.UpdatePost)
		postRoutes.DELETE("/:id", postHandler.DeletePost)
	}
}
