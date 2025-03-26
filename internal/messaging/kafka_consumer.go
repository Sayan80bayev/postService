package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"postService/internal/cache"
	"postService/internal/events"
	"postService/internal/mappers"
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

func (c *KafkaConsumer) Start(ctx context.Context) {
	err := c.consumer.SubscribeTopics(c.config.Topics, nil)
	if err != nil {
		logger.Errorf("Error subscribing to topics: %v", err)
		return
	}

	logger.Info("KafkaConsumer started...")

	for {
		select {
		case <-ctx.Done(): // Graceful shutdown
			logger.Info("KafkaConsumer shutting down...")
			return
		default:
			msg, err := c.consumer.ReadMessage(-1)
			if err == nil {
				logger.Infof("Received message: %s", string(msg.Value))
				c.handleMessage(msg)
			} else {
				logger.Warnf("KafkaConsumer error: %v", err)
			}
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
		logger.Errorf("Error parsing message: %v", err)
		return
	}

	lockKey := "cache:update_lock"
	ctx := context.Background()

	// Try acquiring lock with a reasonable timeout
	locked, err := c.redisClient.SetNX(ctx, lockKey, "1", 10*time.Second).Result()
	if err != nil {
		logger.Errorf("Error acquiring lock: %v", err)
		return
	}
	if !locked {
		logger.Warn("Skipping cache update: another process is still updating the cache.")
		return
	}
	defer c.redisClient.Del(ctx, lockKey) // Ensure lock is released

	// Handle event types
	switch event.Type {
	case "PostCreated":
		cache.UpdateCache(c.redisClient, c.postRepo)
	case "PostUpdated":
		var postUpdated events.PostUpdated
		if err := json.Unmarshal(event.Data, &postUpdated); err == nil {
			c.handlePostUpdated(postUpdated)
		} else {
			logger.Errorf("Error parsing PostUpdated event: %v", err)
		}
	case "PostDeleted":
		var postDeleted events.PostDeleted
		if err := json.Unmarshal(event.Data, &postDeleted); err == nil {
			c.handlePostDeleted(postDeleted)
		} else {
			logger.Errorf("Error parsing PostDeleted event: %v", err)
		}
	default:
		logger.Warnf("Unknown event type: %s", event.Type)
	}
}

func (c *KafkaConsumer) handlePostUpdated(event events.PostUpdated) {
	postID := event.PostID
	newURL := event.FileURL
	oldURL := event.OldURL

	// Delete old file if it was replaced
	if oldURL != "" && oldURL != newURL {
		if err := storage.DeleteFileByURL(oldURL, c.minioClient); err != nil {
			logger.Errorf("Error deleting old file: %v", err)
		}
	}

	// Remove outdated post from cache
	if err := c.redisClient.Del(context.Background(), fmt.Sprintf("post:%d", postID)).Err(); err != nil {
		logger.Errorf("Error deleting post from Redis: %v", err)
	}

	// Fetch updated post from DB
	post, err := c.postRepo.GetPostByID(postID)
	if err != nil {
		logger.Errorf("Error fetching updated post: %v", err)
		return
	}

	// Convert to DTO and update cache
	postResponse := mappers.MapPostToResponse(*post)
	jsonData, err := json.Marshal(postResponse)
	if err != nil {
		logger.Warnf("Could not marshal JSON for post %d: %v", postID, err)
		return
	}

	if err := c.redisClient.Set(context.Background(), fmt.Sprintf("post:%d", postID), jsonData, 5*time.Minute).Err(); err != nil {
		logger.Errorf("Failed to update cache for post %d: %v", postID, err)
	}
	logger.Infof("Successfully updated cache for post: %d", postID)
}

func (c *KafkaConsumer) handlePostDeleted(event events.PostDeleted) {
	postID := event.PostID
	imageURL := event.ImageURL

	// Delete image from MinIO if it exists
	if imageURL != "" {
		if err := storage.DeleteFileByURL(imageURL, c.minioClient); err != nil {
			logger.Errorf("Error deleting image from MinIO: %v", err)
		}
	}

	// Remove post from cache
	if err := c.redisClient.Del(context.Background(), fmt.Sprintf("post:%d", postID)).Err(); err != nil {
		logger.Errorf("Error deleting post from Redis: %v", err)
	}

	logger.Infof("Post %d deleted", postID)
	cache.UpdateCache(c.redisClient, c.postRepo)
}
