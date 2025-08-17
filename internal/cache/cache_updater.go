package cache

import (
	"context"
	"encoding/json"
	"github.com/Sayan80bayev/go-project/pkg/logging"
	"github.com/redis/go-redis/v9"
	"postService/internal/mappers"
	"postService/internal/model"
	"time"
)

var logger = logging.GetLogger()

type PostCacheRepository interface {
	GetPosts() ([]model.Post, error)
}

func UpdateCache(redis *redis.Client, repo PostCacheRepository) {
	ctx := context.Background()
	mapper := mappers.PostMapper{MapFunc: mappers.MapPostToResponse}

	posts, err := repo.GetPosts()
	if err != nil {
		logger.Warn("Error loading posts:", err)
		return
	}

	postResponses := mapper.MapEach(posts)
	jsonData, err := json.Marshal(postResponses)
	if err != nil {
		logger.Warnf("Could not marshal json on update cache: %v", err)
		return
	}

	redis.Set(ctx, "posts:list", jsonData, 5*time.Minute)
	logger.Info("✅ Cache updated successfully!")
}
