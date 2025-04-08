package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"postService/internal/pkg/storage"

	migrateps "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"postService/internal/config"
	"postService/internal/messaging"
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

	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatal("Error getting generic DB object:", err)
		return nil, err
	}

	driver, err := migrateps.WithInstance(sqlDB, &migrateps.Config{})
	if err != nil {
		logger.Fatal("Error initializing migration driver:", err)
		return nil, err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		logger.Fatal("Error creating migration instance:", err)
		return nil, err
	}

	version, _, err := m.Version()
	if err != nil {
		logger.Fatal("Error checking migration version:", err)
		return nil, err
	}

	if version == 0 {
		logger.Info("⚡ Applying migrations for the first time")
	} else {
		logger.Info("⚡ Database already migrated (version:", version, ")")
	}

	if version == 1 {
		logger.Warn("⚠️ Database is in dirty state, forcing migration to version 1")
		if err := m.Force(1); err != nil {
			logger.Fatal("Error forcing migration to version 1:", err)
			return nil, err
		}
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Fatal("Migration failed:", err)
		return nil, err
	}
	logger.Info("✅ Migrations applied successfully")

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
