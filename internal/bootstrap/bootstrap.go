package bootstrap

import (
	"context"
	"fmt"
	"github.com/Sayan80bayev/go-project/pkg/caching"
	"github.com/Sayan80bayev/go-project/pkg/logging"
	"github.com/Sayan80bayev/go-project/pkg/messaging"
	storage "github.com/Sayan80bayev/go-project/pkg/objectStorage"
	"postService/internal/config"
	"postService/internal/repository"
	"postService/internal/service"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Container struct {
	DB       *mongo.Database
	Redis    caching.CacheService
	Minio    storage.FileStorage
	Producer messaging.Producer
	Consumer messaging.Consumer
	Config   *config.Config
	JWKSURL  string
}

func Init() (*Container, error) {
	logger := logging.GetLogger()

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("Error loading configuration:", err)
		return nil, err
	}

	// Собираем JWKS URL
	jwksURL := buildJWKSURL(cfg)

	// Инициализация MongoDB
	db, err := initMongoDatabase(cfg)
	if err != nil {
		return nil, err
	}

	// Инициализация Redis
	cacheService, err := initRedis(cfg)
	if err != nil {
		return nil, err
	}

	// Kafka producer
	producer, err := messaging.NewKafkaProducer(cfg.KafkaBrokers[0], "posts-events")
	if err != nil {
		return nil, err
	}

	fileStorage, err := initMinio(cfg)
	if err != nil {
		return nil, err
	}

	// Kafka consumer
	postRepository := repository.GetPostRepository(db)
	consumer, err := initKafkaConsumer(cacheService, fileStorage, postRepository, cfg)
	if err != nil {
		return nil, err
	}

	logger.Info("✅ Dependencies initialized successfully")

	return &Container{
		DB:       db,
		Redis:    cacheService,
		Producer: producer,
		Consumer: consumer,
		Config:   cfg,
		JWKSURL:  jwksURL,
	}, nil
}

// buildJWKSURL собирает полный путь до JWKS эндпоинта Keycloak
func buildJWKSURL(cfg *config.Config) string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs", cfg.KeycloakURL, cfg.KeycloakRealm)
}

func initMongoDatabase(cfg *config.Config) (*mongo.Database, error) {
	logger := logging.GetLogger()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(cfg.MongoURI)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		logger.Fatal("Error connecting to MongoDB:", err)
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		logger.Fatal("MongoDB ping failed:", err)
		return nil, err
	}

	logger.Info("Connected to MongoDB")

	return client.Database(cfg.MongoDBName), nil
}

func initRedis(cfg *config.Config) (*caching.RedisService, error) {
	logger := logging.GetLogger()
	redisCache, err := caching.NewRedisService(caching.RedisConfig{
		DB:       0,
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
	})

	if err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	logger.Info("Redis connected")
	return redisCache, nil
}

func initMinio(cfg *config.Config) (storage.FileStorage, error) {
	logger := logging.GetLogger()

	minioCfg := &storage.MinioConfig{
		Bucket:    cfg.MinioBucket,
		Host:      cfg.MinioHost,
		AccessKey: cfg.AccessKey,
		SecretKey: cfg.SecretKey,
		Port:      cfg.MinioPort,
	}

	fs, err := storage.NewMinioStorage(minioCfg)
	if err != nil {
		return nil, fmt.Errorf("minio init failed: %w", err)
	}

	logger.Infof("Minio connected: bucket=%s host=%s", cfg.MinioBucket, cfg.MinioHost)
	return fs, nil
}

func initKafkaConsumer(cacheService caching.CacheService, fileService storage.FileStorage, postRepo *repository.PostRepositoryImpl, cfg *config.Config) (*messaging.KafkaConsumer, error) {
	logging.GetLogger().Info("Kafka broker: ", cfg.KafkaBrokers[0])
	consumer, err := messaging.NewKafkaConsumer(messaging.ConsumerConfig{
		BootstrapServers: cfg.KafkaBrokers[0],
		GroupID:          "post-group",
		Topics:           []string{"posts-events"},
	})

	if err != nil {
		logging.GetLogger().Fatal("Error initializing Kafka consumer:", err)
		return nil, err
	}

	consumer.RegisterHandler("PostUpdated", service.PostUpdatedHandler(cacheService, fileService, postRepo))
	return consumer, nil
}
