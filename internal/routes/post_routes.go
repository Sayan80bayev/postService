package routes

import (
	caching "github.com/Sayan80bayev/go-project/pkg/caching"
	"github.com/Sayan80bayev/go-project/pkg/middleware"
	storage "github.com/Sayan80bayev/go-project/pkg/objectStorage"
	"github.com/gin-gonic/gin"
	"postService/internal/bootstrap"
	"postService/internal/delivery"
	"postService/internal/repository"
	"postService/internal/service"
)

func SetupPostRoutes(r *gin.Engine, bs *bootstrap.Container) {
	cfg := bs.Config
	minioClient := bs.Minio
	producer := bs.Producer

	postRepo := repository.GetPostRepository(bs.DB)
	cacheService := caching.NewCacheService(bs.Redis)
	fileStorage := storage.NewMinioStorage(minioClient, &storage.MinioConfig{
		Bucket:    cfg.MinioBucket,
		Host:      cfg.MinioHost,
		AccessKey: cfg.AccessKey,
		SecretKey: cfg.SecretKey,
		Port:      cfg.MinioPort,
	})

	postService := service.NewPostService(postRepo, fileStorage, cacheService, producer)
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
