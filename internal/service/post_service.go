package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/net/context"
	"log"
	"mime/multipart"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"

	"postService/internal/config"
	"postService/internal/mappers"
	"postService/internal/models"
	"postService/internal/repository"
	"postService/internal/response"
	"postService/pkg/s3"
)

type PostService struct {
	postRepo *repository.PostRepository
	minio    *minio.Client
	redis    *redis.Client
	producer *kafka.Producer
	mapper   *mappers.PostMapper
}

func NewPostService(postRepo *repository.PostRepository, minioClient *minio.Client, redis *redis.Client, producer *kafka.Producer) *PostService {
	return &PostService{
		postRepo: postRepo,
		minio:    minioClient,
		redis:    redis,
		producer: producer,
		mapper:   mappers.NewPostMapper(),
	}
}

// ✅ Создание поста
func (ps *PostService) CreatePost(title, content string, userID uint, categoryID uint, file multipart.File, header *multipart.FileHeader, cfg *config.Config) error {
	var imageURL string
	if file != nil && header != nil {
		uploadedURL, err := s3.UploadFile(file, header, cfg, ps.minio)
		if err != nil {
			return err
		}
		imageURL = uploadedURL
	}

	post := &models.Post{
		Title:      title,
		Content:    content,
		UserID:     userID,
		CategoryID: categoryID,
		ImageURL:   imageURL,
		LikeCount:  0,
	}

	if err := ps.postRepo.CreatePost(post); err != nil {
		return err
	}

	ps.redis.Publish(context.TODO(), "posts:updates", "update")
	return nil
}

// ✅ Получение списка постов (кеширование в Redis)
func (ps *PostService) GetPosts() ([]response.PostResponse, error) {
	ctx := context.TODO()
	cachedPosts, err := ps.redis.Get(ctx, "posts:list").Result()
	if err == nil {
		var postResponses []response.PostResponse
		json.Unmarshal([]byte(cachedPosts), &postResponses)
		return postResponses, nil
	}

	posts, err := ps.postRepo.GetPosts()
	if err != nil {
		return nil, err
	}

	postResponses := ps.mapper.MapEach(posts)
	jsonData, _ := json.Marshal(postResponses)
	ps.redis.Set(ctx, "posts:list", jsonData, 10*time.Minute)

	return postResponses, nil
}

// ✅ Получение поста по ID (кеширование)
func (ps *PostService) GetPostByID(id uint) (*response.PostResponse, error) {
	ctx := context.TODO()
	cacheKey := fmt.Sprintf("post:%d", id)

	cachedPost, err := ps.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var post response.PostResponse
		json.Unmarshal([]byte(cachedPost), &post)
		return &post, nil
	}

	post, err := ps.postRepo.GetPostByID(id)
	if err != nil {
		return nil, err
	}

	resp := ps.mapper.Map(*post)
	jsonData, _ := json.Marshal(resp)
	ps.redis.Set(ctx, cacheKey, jsonData, 10*time.Minute)

	return &resp, nil
}

// ✅ Обновление поста (фикс обновления категории)
func (ps *PostService) UpdatePost(content string, title string, userId uint, postId uint, categoryID uint, file multipart.File, header *multipart.FileHeader, cfg *config.Config) error {
	post, err := validatePermission(userId, postId, ps)
	if err != nil {
		return err
	}

	// Загружаем новое изображение, если передано
	if file != nil && header != nil {
		imageURL, err := s3.UploadFile(file, header, cfg, ps.minio)
		if err != nil {
			return err
		}
		post.ImageURL = imageURL
	}

	// Обновляем поля
	post.Content = content
	post.Title = title
	post.CategoryID = categoryID // ✅ Фикс обновления категории

	err = ps.postRepo.UpdatePost(post)
	if err != nil {
		return err
	}

	ctx := context.TODO()
	cacheKey := fmt.Sprintf("post:%d", postId)
	ps.redis.Del(ctx, cacheKey)
	ps.redis.Publish(ctx, "posts:updates", "update")

	return nil
}

// ✅ Удаление поста
func (ps *PostService) DeletePost(postId uint, userId uint) error {
	_, err := validatePermission(userId, postId, ps)
	if err != nil {
		return err
	}

	err = ps.postRepo.DeletePost(postId)
	if err != nil {
		return err
	}

	ctx := context.TODO()
	cacheKey := fmt.Sprintf("post:%d", postId)
	log.Println("Delete post:", postId)
	ps.redis.Del(ctx, cacheKey)
	ps.redis.Publish(ctx, "posts:updates", "update")

	return nil
}

// ✅ Валидация перед обновлением
func validatePermission(userId uint, postId uint, ps *PostService) (*models.Post, error) {
	post, err := ps.postRepo.GetPostByID(postId)
	if err != nil {
		return nil, errors.New("post not found")
	}
	if post.User.ID != userId {
		return nil, errors.New("user not allowed")
	}
	return post, nil
}
