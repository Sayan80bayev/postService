package bootstrap

import (
	"context"
	"time"

	"postService/internal/config"
	"postService/internal/messaging"
	"postService/internal/pkg/storage"
	"postService/internal/repository"
	"postService/pkg/logging"

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
}

func Init() (*Container, error) {
	logger := logging.GetLogger()

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("Error loading configuration:", err)
		return nil, err
	}

	db, err := initMongoDatabase(cfg)
	if err != nil {
		return nil, err
	}

	redisClient, err := initRedis(cfg)
	if err != nil {
		return nil, err
	}

	minioClient := storage.Init(cfg)

	producer, err := messaging.NewKafkaProducer(cfg.KafkaBrokers[0], "posts-events")
	if err != nil {
		logger.Fatal("Error creating Kafka producer:", err)
		return nil, err
	}

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
	}, nil
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
