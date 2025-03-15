package routes

import (
	"github.com/gin-gonic/gin"
	"postService/internal/bootstrap"
	"postService/internal/delivery"
	"postService/internal/repository"
	"postService/internal/service"
)

func SetupPostRoutes(r *gin.Engine, bs *bootstrap.Bootstrap, authMiddleware gin.HandlerFunc) {
	cfg := bs.Config
	minioClient := bs.Minio
	producer := bs.Producer

	repoInterface, err := bs.GetRepository("post")
	if err != nil {
		logger.Error("Failed to get post repository: " + err.Error())
	}
	postRepo, ok := repoInterface.(*repository.PostRepository)
	if !ok {
		logger.Error("Invalid repository type for post")
	}

	cacheService := service.NewCacheService(bs.Redis)

	postService := service.NewPostService(postRepo, minioClient, cacheService, producer)
	postHandler := delivery.NewPostHandler(postService, cfg)

	r.GET("api/v1/posts", postHandler.GetPosts)
	r.GET("api/v1/posts/:id", postHandler.GetPostByID)

	postRoutes := r.Group("api/v1/posts", authMiddleware)
	{
		postRoutes.POST("/", postHandler.CreatePost)
		postRoutes.PUT("/:id", postHandler.UpdatePost)
		postRoutes.DELETE("/:id", postHandler.DeletePost)
	}
}
