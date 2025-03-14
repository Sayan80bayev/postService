package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"postService/internal/config"
	"postService/internal/delivery"
	"postService/internal/messaging"
	"postService/internal/repository"
	"postService/internal/service"
)

// SetupPostRoutes настраивает маршруты для работы с постами
func SetupPostRoutes(r *gin.Engine, db *gorm.DB, authMiddleware gin.HandlerFunc, client *redis.Client, producer *messaging.Producer, minioClient *minio.Client, cfg *config.Config) {

	postRepo := repository.NewPostRepository(db)
	postService := service.NewPostService(postRepo, minioClient, client, producer)
	postHandler := delivery.NewPostHandler(postService, cfg)

	// Открытые роуты
	r.GET("api/v1/posts", postHandler.GetPosts)
	r.GET("api/v1/posts/:id", postHandler.GetPostByID)

	// Защищённые роуты (требуется авторизация)
	postRoutes := r.Group("api/v1/posts", authMiddleware)
	{
		postRoutes.POST("/", postHandler.CreatePost)
		postRoutes.PUT("/:id", postHandler.UpdatePost)
		postRoutes.DELETE("/:id", postHandler.DeletePost)
	}
}
