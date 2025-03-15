package cache

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"postService/internal/mappers"
	"postService/internal/repository"
	"postService/pkg/logging"
	"time"
)

var logger = logging.GetLogger()

func UpdateCache(redis *redis.Client, postRepo *repository.PostRepository) {
	ctx := context.Background()
	mapper := mappers.PostMapper{MapFunc: mappers.MapPostToResponse}

	posts, err := postRepo.GetPosts()
	if err != nil {
		logger.Warn("Error loading posts:", err)
		return
	}

	postResponses := mapper.MapEach(posts)
	jsonData, _ := json.Marshal(postResponses)

	redis.Set(ctx, "posts:list", jsonData, 5*time.Minute)
	logger.Info("✅ Cache updated successfully!")
}
