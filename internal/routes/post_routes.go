package routes

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"postService/internal/cache"
	"postService/internal/config"
	"postService/internal/delivery"
	"postService/internal/repository"
	"postService/internal/service"
	"postService/pkg/s3"
)

// SetupPostRoutes настраивает маршруты для работы с постами
func SetupPostRoutes(r *gin.Engine, db *gorm.DB, authMiddleware gin.HandlerFunc, client *redis.Client, producer *kafka.Producer, cfg *config.Config) {

	minioClient := s3.Init(cfg)
	postRepo := repository.NewPostRepository(db)
	postService := service.NewPostService(postRepo, minioClient, client, producer)
	postHandler := delivery.NewPostHandler(postService, cfg)

	cacheListener := cache.NewCacheListener(client, postRepo)
	go cacheListener.ListenForPostUpdates()

	// Открытые роуты
	r.GET("/posts", postHandler.GetPosts)
	r.GET("/posts/:id", postHandler.GetPostByID)

	// Защищённые роуты (требуется авторизация)
	postRoutes := r.Group("/posts", authMiddleware)
	{
		postRoutes.POST("/", postHandler.CreatePost)
		postRoutes.PUT("/:id", postHandler.UpdatePost)
		postRoutes.DELETE("/:id", postHandler.DeletePost)
	}
}
