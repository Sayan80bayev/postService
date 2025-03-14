package routes

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"postService/internal/config"
	"postService/pkg/middleware"
)

// SetupRoutes подключает все маршруты
func SetupRoutes(r *gin.Engine, db *gorm.DB, client *redis.Client, producer *kafka.Producer, cfg *config.Config) {
	// Middleware аутентификации
	authMiddleware := middleware.AuthMiddleware(cfg.JWTSecret)

	// Подключаем отдельные роутеры
	SetupPostRoutes(r, db, authMiddleware, client, producer, cfg)
	SetupCategoryRoutes(r, db, authMiddleware)
}
