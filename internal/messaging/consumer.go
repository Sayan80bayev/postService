package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"postService/internal/cache"
	"postService/internal/events"
	"postService/internal/repository"
	"postService/pkg/s3"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
)

// ConsumerConfig holds configuration for the Kafka consumer
type ConsumerConfig struct {
	BootstrapServers string
	GroupID          string
	Topics           []string
}

// Consumer represents a Kafka consumer
type Consumer struct {
	config      ConsumerConfig
	consumer    *kafka.Consumer
	redisClient *redis.Client
	minioClient *minio.Client
	postRepo    *repository.PostRepository
}

// NewConsumer initializes a new Kafka consumer
func NewConsumer(config ConsumerConfig, redisClient *redis.Client, minioClient *minio.Client, postRepo *repository.PostRepository) (*Consumer, error) {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": config.BootstrapServers,
		"group.id":          config.GroupID,
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		return nil, err
	}

	return &Consumer{
		config:      config,
		consumer:    consumer,
		redisClient: redisClient,
		minioClient: minioClient,
		postRepo:    postRepo,
	}, nil
}

// Start consumes messages from Kafka topics
func (c *Consumer) Start() {
	err := c.consumer.SubscribeTopics(c.config.Topics, nil)
	if err != nil {
		fmt.Println("Error subscribing to topics:", err)
		return
	}

	fmt.Println("Kafka Consumer started...")

	for {
		msg, err := c.consumer.ReadMessage(-1)
		if err == nil {
			fmt.Printf("Received message: %s\n", string(msg.Value))
			c.handleMessage(msg) // Вызов обработчика сообщений
		} else {
			fmt.Printf("Consumer error: %v\n", err)
		}
	}
}

// Close shuts down the consumer
func (c *Consumer) Close() {
	c.consumer.Close()
}

// handleMessage processes incoming Kafka messages
func (c *Consumer) handleMessage(msg *kafka.Message) {
	var event events.Event
	err := json.Unmarshal(msg.Value, &event)
	if err != nil {
		log.Printf("Error parsing message: %v", err)
		return
	}

	switch event.Type {
	case "PostUpdated":
		var postUpdated events.PostUpdated
		if err := json.Unmarshal(event.Data, &postUpdated); err == nil {
			c.handlePostUpdated(postUpdated)
		}
	case "PostDeleted":
		var postDeleted events.PostDeleted
		if err := json.Unmarshal(event.Data, &postDeleted); err == nil {
			c.handlePostDeleted(postDeleted)
		}
	default:
		log.Println("Unknown event type:", event.Type)
	}
}

// handlePostUpdated handles PostUpdated event
func (c *Consumer) handlePostUpdated(event events.PostUpdated) {
	postID := event.PostID
	newURL := event.FileURL
	oldURL := event.OldURL

	// Удаление старого файла, если URL изменился
	if oldURL != "" && oldURL != newURL {
		err := s3.DeleteFileByURL(oldURL, c.minioClient)
		if err != nil {
			log.Println("Ошибка удаления старого файла:", err)
		}
	}

	// Удаление кэша поста
	err := c.redisClient.Del(context.Background(), "post:"+fmt.Sprint(postID)).Err()
	if err != nil {
		log.Println("Ошибка удаления поста из Redis:", err)
	}
	log.Println("Successfully updated post:", postID)

	// Обновление кэша
	cache.UpdateCache(c.redisClient, c.postRepo)
}

// handlePostDeleted handles PostDeleted event
func (c *Consumer) handlePostDeleted(event events.PostDeleted) {
	postID := event.PostID
	imageURL := event.ImageURL

	// Удаление изображения из MinIO
	if imageURL != "" {
		err := s3.DeleteFileByURL(imageURL, c.minioClient)
		if err != nil {
			log.Println("Ошибка удаления изображения из MinIO:", err)
		}
	}

	// Удаление поста из Redis
	err := c.redisClient.Del(context.Background(), "post:"+fmt.Sprint(postID)).Err()
	if err != nil {
		log.Println("Ошибка удаления поста из Redis:", err)
	}

	log.Println("Post image is deleted")

	// Обновление кэша
	cache.UpdateCache(c.redisClient, c.postRepo)
}
