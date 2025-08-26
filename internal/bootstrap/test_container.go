package bootstrap

import (
	"fmt"
	"github.com/Sayan80bayev/go-project/pkg/messaging"
	"postService/internal/config"
	"postService/internal/repository"
)

func NewTestContainer(mongoURI, kafkaAddr, minioHost, minioPort, redisAddr, jwksURL string) *Container {
	cfg := &config.Config{
		MongoURI:     mongoURI,
		MongoDBName:  "testdb",
		RedisAddr:    redisAddr,
		RedisPass:    "",
		MinioBucket:  "test-bucket",
		MinioHost:    minioHost,
		MinioPort:    minioPort, // usually testcontainer default
		AccessKey:    "admin",
		SecretKey:    "admin123",
		KafkaBrokers: []string{kafkaAddr},
		KafkaTopic:   "post-events",
	}

	// Mongo
	db, err := initMongoDatabase(cfg)
	if err != nil {
		panic(err)
	}

	// Redis
	cs, err := initRedis(cfg)
	if err != nil {
		panic(err)
	}

	// MinIO
	fs, err := initMinio(cfg)
	if err != nil {
		panic(err)
	}

	// Kafka Producer
	producer, err := messaging.NewKafkaProducer(cfg.KafkaBrokers[0], cfg.KafkaTopic)
	if err != nil {
		panic(fmt.Errorf("failed to create Kafka producer: %w", err))
	}

	ur := repository.NewPostRepository(db)

	// Kafka Consumer
	consumer, err := initKafkaConsumer(cs, fs, ur, cfg)
	if err != nil {
		panic(err)
	}
	// Use typed event constants

	return &Container{
		DB:             db,
		Redis:          cs,
		FileStorage:    fs,
		Producer:       producer,
		Consumer:       consumer,
		PostRepository: ur,
		Config:         cfg,
		JWKSurl:        jwksURL,
	}
}
