package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"postService/internal/cache"
	"postService/internal/events"
	"postService/internal/pkg/storage"
	"postService/internal/repository"
	"postService/pkg/logging"
)

type ConsumerConfig struct {
	BootstrapServers string
	GroupID          string
	Topics           []string
}

type KafkaConsumer struct {
	config      ConsumerConfig
	consumer    *kafka.Consumer
	redisClient *redis.Client
	minioClient *minio.Client
	postRepo    *repository.PostRepositoryImpl
}

func NewKafkaConsumer(config ConsumerConfig, redisClient *redis.Client, minioClient *minio.Client, postRepo *repository.PostRepositoryImpl) (*KafkaConsumer, error) {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": config.BootstrapServers,
		"group.id":          config.GroupID,
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		return nil, err
	}

	return &KafkaConsumer{
		config:      config,
		consumer:    consumer,
		redisClient: redisClient,
		minioClient: minioClient,
		postRepo:    postRepo,
	}, nil
}

var logger = logging.GetLogger()

func (c *KafkaConsumer) Start() {
	err := c.consumer.SubscribeTopics(c.config.Topics, nil)
	if err != nil {
		logger.Errorf("Error subscribing to topics: %v", err)
		return
	}

	logger.Info("Kafka KafkaConsumer started...")

	for {
		msg, err := c.consumer.ReadMessage(-1)
		if err == nil {
			logger.Infof("Received message: %s", string(msg.Value))
			c.handleMessage(msg)
		} else {
			logger.Warnf("KafkaConsumer error: %v", err)
		}
	}
}

func (c *KafkaConsumer) Close() {
	c.consumer.Close()
}

func (c *KafkaConsumer) handleMessage(msg *kafka.Message) {
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

func (c *KafkaConsumer) handlePostUpdated(event events.PostUpdated) {
	postID := event.PostID
	newURL := event.FileURL
	oldURL := event.OldURL

	if oldURL != "" && oldURL != newURL {
		err := storage.DeleteFileByURL(oldURL, c.minioClient)
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

func (c *KafkaConsumer) handlePostDeleted(event events.PostDeleted) {
	postID := event.PostID
	imageURL := event.ImageURL

	if imageURL != "" {
		err := storage.DeleteFileByURL(imageURL, c.minioClient)
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
