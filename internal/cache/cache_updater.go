package cache

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"log"
	"postService/internal/mappers"
	"postService/internal/repository"
	"time"
)

func UpdateCache(redis *redis.Client, postRepo *repository.PostRepository) {
	ctx := context.Background()
	mapper := mappers.PostMapper{MapFunc: mappers.MapPostToResponse}

	posts, err := postRepo.GetPosts()
	if err != nil {
		log.Println("Ошибка загрузки постов:", err)
		return
	}

	postResponses := mapper.MapEach(posts)
	jsonData, _ := json.Marshal(postResponses)

	redis.Set(ctx, "posts:list", jsonData, 5*time.Minute)
	log.Println("✅ Кэш обновлён!")
}
