package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"postService/internal/cache"
	"postService/internal/events"
	"postService/internal/repository"
	"postService/pkg/logging"
	"postService/pkg/s3"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
)

type ConsumerConfig struct {
	BootstrapServers string
	GroupID          string
	Topics           []string
}

type Consumer struct {
	config      ConsumerConfig
	consumer    *kafka.Consumer
	redisClient *redis.Client
	minioClient *minio.Client
	postRepo    *repository.PostRepository
}

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

var logger = logging.GetLogger()

func (c *Consumer) Start() {
	err := c.consumer.SubscribeTopics(c.config.Topics, nil)
	if err != nil {
		logger.Errorf("Error subscribing to topics: %v", err)
		return
	}

	logger.Info("Kafka Consumer started...")

	for {
		msg, err := c.consumer.ReadMessage(-1)
		if err == nil {
			logger.Infof("Received message: %s", string(msg.Value))
			c.handleMessage(msg)
		} else {
			logger.Warnf("Consumer error: %v", err)
		}
	}
}

func (c *Consumer) Close() {
	c.consumer.Close()
}

func (c *Consumer) handleMessage(msg *kafka.Message) {
	var event events.Event
	err := json.Unmarshal(msg.Value, &event)
	if err != nil {
		logger.Errorf("Error parsing message: %v", err)
		return
	}

	switch event.Type {
	case "PostCreated":
		cache.UpdateCache(c.redisClient, c.postRepo)
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
		logger.Warnf("Unknown event type: %s", event.Type)
	}
}

func (c *Consumer) handlePostUpdated(event events.PostUpdated) {
	postID := event.PostID
	newURL := event.FileURL
	oldURL := event.OldURL

	if oldURL != "" && oldURL != newURL {
		err := s3.DeleteFileByURL(oldURL, c.minioClient)
		if err != nil {
			logger.Errorf("Error deleting old file: %v", err)
		}
	}

	err := c.redisClient.Del(context.Background(), "post:"+fmt.Sprint(postID)).Err()
	if err != nil {
		logger.Errorf("Error deleting post from Redis: %v", err)
	}

	logger.Infof("Successfully updated post: %d", postID)

	cache.UpdateCache(c.redisClient, c.postRepo)
}

func (c *Consumer) handlePostDeleted(event events.PostDeleted) {
	postID := event.PostID
	imageURL := event.ImageURL

	if imageURL != "" {
		err := s3.DeleteFileByURL(imageURL, c.minioClient)
		if err != nil {
			logger.Errorf("Error deleting image from MinIO: %v", err)
		}
	}

	err := c.redisClient.Del(context.Background(), "post:"+fmt.Sprint(postID)).Err()
	if err != nil {
		logger.Errorf("Error deleting post from Redis: %v", err)
	}

	logger.Infof("Post %d deleted", postID)

	cache.UpdateCache(c.redisClient, c.postRepo)
}
