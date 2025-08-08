package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"postService/internal/storage"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"postService/internal/cache"
	"postService/internal/events"
	"postService/internal/mappers"
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

func (c *KafkaConsumer) Start(ctx context.Context) {
	if err := c.consumer.SubscribeTopics(c.config.Topics, nil); err != nil {
		logger.Errorf("Error subscribing to topics: %v", err)
		return
	}

	logger.Info("KafkaConsumer started...")

	for {
		select {
		case <-ctx.Done():
			logger.Info("KafkaConsumer shutting down...")
			return
		default:
			msg, err := c.consumer.ReadMessage(-1)
			if err != nil {
				logger.Warnf("KafkaConsumer error: %v", err)
				continue
			}
			logger.Infof("Received message: %s", string(msg.Value))
			c.handleMessage(msg)
		}
	}
}

func (c *KafkaConsumer) Close() {
	logger.Info("Closing Kafka consumer...")
	if err := c.consumer.Close(); err != nil {
		logger.Errorf("Could not close Kafka consumer: %v", err)
	} else {
		logger.Info("Kafka consumer closed successfully")
	}
}

func (c *KafkaConsumer) handleMessage(msg *kafka.Message) {
	var event events.Event
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		logger.Errorf("Error parsing event envelope: %v", err)
		return
	}

	ctx := context.Background()
	lockKey := "cache:update_lock"

	locked, err := c.redisClient.SetNX(ctx, lockKey, "1", 10*time.Second).Result()
	if err != nil {
		logger.Errorf("Error acquiring Redis lock: %v", err)
		return
	}
	if !locked {
		logger.Warn("Skipping message handling: another process holds the lock.")
		return
	}
	defer c.redisClient.Del(ctx, lockKey)

	switch event.Type {
	case "PostCreated":
		cache.UpdateCache(c.redisClient, c.postRepo)

	case "PostUpdated":
		var data events.PostUpdated
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Errorf("Error parsing PostUpdated: %v", err)
			return
		}
		c.handlePostUpdated(data)

	case "PostDeleted":
		var data events.PostDeleted
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Errorf("Error parsing PostDeleted: %v", err)
			return
		}
		c.handlePostDeleted(data)

	default:
		logger.Warnf("Unknown event type: %s", event.Type)
	}
}

func (c *KafkaConsumer) handlePostUpdated(event events.PostUpdated) {
	ctx := context.Background()

	// Delete old files from MinIO
	for _, oldURL := range event.OldURLs {
		if err := storage.DeleteFileByURL(oldURL, c.minioClient); err != nil {
			logger.Warnf("Failed to delete old file: %s, error: %v", oldURL, err)
		}
	}

	if err := c.redisClient.Del(ctx, fmt.Sprintf("post:%s", event.PostID)).Err(); err != nil {
		logger.Errorf("Error deleting post from Redis: %v", err)
	}

	// Refresh cache
	post, err := c.postRepo.GetPostByID(event.PostID)
	if err != nil {
		logger.Errorf("Error fetching post by ID: %v", err)
		return
	}

	postResponse := mappers.MapPostToResponse(*post)
	jsonData, err := json.Marshal(postResponse)
	if err != nil {
		logger.Warnf("Could not marshal post to JSON: %v", err)
		return
	}

	if err := c.redisClient.Set(ctx, fmt.Sprintf("post:%s", event.PostID), jsonData, 5*time.Minute).Err(); err != nil {
		logger.Errorf("Failed to update Redis cache: %v", err)
		return
	}

	logger.Infof("Post %s cache updated", event.PostID)
}

func (c *KafkaConsumer) handlePostDeleted(event events.PostDeleted) {
	ctx := context.Background()

	// Delete all image and file URLs from MinIO
	for _, url := range append(event.ImageURLs, event.FileURLs...) {
		if err := storage.DeleteFileByURL(url, c.minioClient); err != nil {
			logger.Warnf("Error deleting file/image from MinIO (%s): %v", url, err)
		}
	}

	// Delete from Redis
	if err := c.redisClient.Del(ctx, fmt.Sprintf("post:%s", event.PostID)).Err(); err != nil {
		logger.Errorf("Error deleting post from Redis: %v", err)
	}

	logger.Infof("Post %s deleted and cleaned up", event.PostID)

	// Rebuild full cache (optional depending on system strategy)
	cache.UpdateCache(c.redisClient, c.postRepo)
}
