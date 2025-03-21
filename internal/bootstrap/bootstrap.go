package bootstrap

import (
	"context"
	"fmt"
	"postService/internal/pkg/storage"

	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"postService/internal/config"
	"postService/internal/messaging"
	"postService/internal/model"
	"postService/internal/repository"
	"postService/pkg/logging"
)

type Container struct {
	DB           *gorm.DB
	Redis        *redis.Client
	Minio        *minio.Client
	Producer     messaging.Producer
	Consumer     messaging.Consumer
	Config       *config.Config
	Repositories map[string]interface{}
}

func Init() (*Container, error) {
	logger := logging.GetLogger()

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("Error loading configuration:", err)
		return nil, err
	}

	// Инициализация зависимостей
	db, err := initDatabase(cfg)
	if err != nil {
		return nil, err
	}

	redisClient, err := initRedis(cfg)
	if err != nil {
		return nil, err
	}

	minioClient := storage.Init(cfg)

	producer, err := messaging.NewKafkaProducer("localhost:9092", "posts-events")
	if err != nil {
		logger.Fatal("Error creating Kafka KafkaProducer:", err)
		return nil, err
	}

	repositories := initRepositories(db)

	consumer, err := initKafkaConsumer(redisClient, minioClient, repositories["post"].(*repository.PostRepositoryImpl))
	if err != nil {
		return nil, err
	}

	logger.Info("✅ Dependencies initialized successfully")

	return &Container{
		DB:           db,
		Redis:        redisClient,
		Minio:        minioClient,
		Producer:     producer,
		Consumer:     consumer,
		Config:       cfg,
		Repositories: repositories,
	}, nil
}

func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	logger := logging.GetLogger()

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		logger.Fatal("Error connecting to the database:", err)
		return nil, err
	}

	err = db.AutoMigrate(&model.Post{}, &model.Category{})
	if err != nil {
		logger.Fatal("Database migration error:", err)
		return nil, err
	}

	return db, nil
}

func initRedis(cfg *config.Config) (*redis.Client, error) {
	logger := logging.GetLogger()

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       0,
	})

	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		logger.Fatal("Error connecting to Redis:", err)
		return nil, err
	}

	return client, nil
}

func initRepositories(db *gorm.DB) map[string]interface{} {
	return map[string]interface{}{
		"post":     repository.NewPostRepository(db),
		"category": repository.NewCategoryRepository(db),
	}
}

func initKafkaConsumer(redisClient *redis.Client, minioClient *minio.Client, postRepo *repository.PostRepositoryImpl) (*messaging.KafkaConsumer, error) {
	consumer, err := messaging.NewKafkaConsumer(messaging.ConsumerConfig{
		BootstrapServers: "localhost:9092",
		GroupID:          "post-group",
		Topics:           []string{"posts-events"},
	}, redisClient, minioClient, postRepo)

	if err != nil {
		logging.GetLogger().Fatal("Error initializing Kafka KafkaConsumer:", err)
		return nil, err
	}

	return consumer, nil
}

func (b *Container) GetRepository(name string) (interface{}, error) {
	repo, exists := b.Repositories[name]
	if !exists {
		return nil, fmt.Errorf("repository %s not found", name)
	}
	return repo, nil
}
