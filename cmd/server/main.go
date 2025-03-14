package main

import (
	"context"
	"log"
	"postService/internal/messaging"
	"postService/internal/repository"
	"postService/pkg/s3"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"postService/internal/config"
	"postService/internal/models"
	"postService/internal/routes"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Ошибка загрузки конфигурации:", err)
	}

	// Подключение к БД
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatal("Ошибка подключения к базе данных:", err)
	}

	err = db.AutoMigrate(&models.Post{}, &models.Category{})
	if err != nil {
		log.Fatal("Ошибка миграции в базе данных:", err)
	}

	// Подключение к Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass, // Оставьте пустым, если пароль не нужен
		DB:       0,             // Используем стандартную базу
	})

	// Проверка соединения с Redis
	ctx := context.Background()
	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatal("Ошибка подключения к Redis:", err)
	}

	// Подключение к Kafka
	producer, err := messaging.NewProducer("localhost:9092", "posts-events")
	if err != nil {
		log.Fatal("Ошибка при создании Kafka Producer:", err)
	}
	defer producer.Close()

	minioClient := s3.Init(cfg)
	postRepo := repository.NewPostRepository(db)
	consumer, _ := messaging.NewConsumer(messaging.ConsumerConfig{
		BootstrapServers: "localhost:9092",
		GroupID:          "post-group",
		Topics:           []string{"posts-events"},
	}, redisClient, minioClient, postRepo)

	go consumer.Start()
	log.Println("✅ Подключение к базе данных, Redis и Kafka успешно установлено")

	r := gin.Default()
	routes.SetupRoutes(r, db, redisClient, producer, minioClient, cfg)

	// Запуск сервера
	r.Run(":" + cfg.Port)
}
