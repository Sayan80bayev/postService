package routes

import (
	"github.com/gin-gonic/gin"
	"postService/internal/bootstrap"
	"postService/internal/delivery"
	"postService/internal/middleware"
	"postService/internal/repository"
	"postService/internal/service"
	"postService/internal/storage"
)

func SetupPostRoutes(r *gin.Engine, bs *bootstrap.Container) {
	cfg := bs.Config
	minioClient := bs.Minio
	producer := bs.Producer

	postRepo := repository.GetPostRepository(bs.DB)
	cacheService := service.NewCacheService(bs.Redis)
	fileStorage := storage.NewMinioStorage(minioClient, cfg)
	postService := service.NewPostService(postRepo, fileStorage, cacheService, producer)
	postHandler := delivery.NewPostHandler(postService, cfg)

	// Публичные маршруты
	r.GET("api/v1/posts", postHandler.GetPosts)
	r.GET("api/v1/posts/:id", postHandler.GetPostByID)

	// Защищённые маршруты
	postRoutes := r.Group("api/v1/posts", middleware.AuthMiddleware(bs.JWKSURL))
	{
		postRoutes.POST("/", postHandler.CreatePost)
		postRoutes.PUT("/:id", postHandler.UpdatePost)
		postRoutes.DELETE("/:id", postHandler.DeletePost)
	}
}
