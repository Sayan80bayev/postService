package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"postService/internal/config"
	"postService/internal/messaging"
	"postService/pkg/middleware"
)

// SetupRoutes подключает все маршруты
func SetupRoutes(r *gin.Engine, db *gorm.DB, client *redis.Client, producer *messaging.Producer, minioClient *minio.Client, cfg *config.Config) {
	// Middleware аутентификации
	authMiddleware := middleware.AuthMiddleware(cfg.JWTSecret)

	// Подключаем отдельные роутеры
	SetupPostRoutes(r, db, authMiddleware, client, producer, minioClient, cfg)
	SetupCategoryRoutes(r, db, authMiddleware)
}
