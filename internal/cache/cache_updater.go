package cache

import (
	"context"
	"encoding/json"
	"github.com/Sayan80bayev/go-project/pkg/caching"
	"github.com/Sayan80bayev/go-project/pkg/logging"
	"postService/internal/mappers"
	"postService/internal/model"
	"time"
)

const (
	CacheKeyPostsList = "posts:list"
	CacheKeyPostFmt   = "post:%s"
	CacheTTL          = 10 * time.Minute
)

var logger = logging.GetLogger()

type PostCacheRepository interface {
	GetPosts(ctx context.Context) ([]model.Post, error)
}

func UpdateCache(cacheService caching.CacheService, repo PostCacheRepository) {
	ctx := context.Background()
	mapper := mappers.PostMapper{MapFunc: mappers.MapPostToResponse}

	posts, err := repo.GetPosts(ctx)
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

	if err := cacheService.Set(ctx, CacheKeyPostsList, jsonData, CacheTTL); err != nil {
		logger.Errorf("Failed to set posts:list in cache: %v", err)
		return
	}

	logger.Info("✅ Cache updated successfully!")
}
