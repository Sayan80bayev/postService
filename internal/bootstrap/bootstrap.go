package bootstrap

import (
	"context"
	"fmt"
	"github.com/Sayan80bayev/go-project/pkg/logging"
	storage "github.com/Sayan80bayev/go-project/pkg/objectStorage"
	"postService/internal/config"
	"postService/internal/messaging"
	"postService/internal/repository"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Container struct {
	DB       *mongo.Database
	Redis    *redis.Client
	Minio    *minio.Client
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
	redisClient, err := initRedis(cfg)
	if err != nil {
		return nil, err
	}

	// Инициализация MinIO
	minioClient := storage.Init(&storage.MinioConfig{
		Bucket:    cfg.MinioBucket,
		Host:      cfg.MinioHost,
		AccessKey: cfg.AccessKey,
		SecretKey: cfg.SecretKey,
		Port:      cfg.MinioPort,
	})

	// Kafka producer
	producer, err := messaging.NewKafkaProducer(cfg.KafkaBrokers[0], "posts-events")
	if err != nil {
		logger.Fatal("Error creating Kafka producer:", err)
		return nil, err
	}

	// Kafka consumer
	postRepository := repository.GetPostRepository(db)
	consumer, err := initKafkaConsumer(redisClient, minioClient, postRepository, cfg)
	if err != nil {
		return nil, err
	}

	logger.Info("✅ Dependencies initialized successfully")

	return &Container{
		DB:       db,
		Redis:    redisClient,
		Minio:    minioClient,
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

func initKafkaConsumer(redisClient *redis.Client, minioClient *minio.Client, postRepo *repository.PostRepositoryImpl, cfg *config.Config) (*messaging.KafkaConsumer, error) {
	logging.GetLogger().Info("Kafka broker: ", cfg.KafkaBrokers[0])
	consumer, err := messaging.NewKafkaConsumer(messaging.ConsumerConfig{
		BootstrapServers: cfg.KafkaBrokers[0],
		GroupID:          "post-group",
		Topics:           []string{"posts-events"},
	}, redisClient, minioClient, postRepo)

	if err != nil {
		logging.GetLogger().Fatal("Error initializing Kafka consumer:", err)
		return nil, err
	}

	return consumer, nil
}
