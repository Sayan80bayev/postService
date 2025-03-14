package cache

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
	"postService/internal/mappers"
	"postService/internal/repository"
)

func UpdateCache(redis *redis.Client, postRepo *repository.PostRepository) {
	ctx := context.Background()
	mapper := mappers.PostMapper{MapFunc: mappers.MapPostToResponse}

	posts, err := postRepo.GetPosts()
	if err != nil {
		log.Println("Ошибка загрузки постов:", err)
		return
	}

	// Конвертируем в JSON
	postResponses := mapper.MapEach(posts)
	jsonData, _ := json.Marshal(postResponses)

	// Обновляем кэш
	redis.Set(ctx, "posts:list", jsonData, 0) // 0 – без истечения времени
	log.Println("✅ Кэш обновлён!")
}
