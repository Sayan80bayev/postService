package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	caching "github.com/Sayan80bayev/go-project/pkg/caching"
	"github.com/Sayan80bayev/go-project/pkg/logging"
	"github.com/Sayan80bayev/go-project/pkg/messaging"
	storage "github.com/Sayan80bayev/go-project/pkg/objectStorage"
	"github.com/google/uuid"
	"mime/multipart"
	"path/filepath"
	"postService/internal/events"
	"postService/internal/mappers"
	"postService/internal/model"
	"postService/internal/transfer/request"
	"postService/internal/transfer/response"
	"strings"
	"time"
)

type PostRepository interface {
	CreatePost(ctx context.Context, post *model.Post) error
	GetPosts(ctx context.Context) ([]model.Post, error)
	GetPostByID(ctx context.Context, id uuid.UUID) (*model.Post, error)
	UpdatePost(ctx context.Context, post *model.Post) error
	DeletePost(ctx context.Context, id uuid.UUID) error
}

type PostService struct {
	repo         PostRepository
	cacheService caching.CacheService
	fileStorage  storage.FileStorage
	producer     messaging.Producer
	mapper       *mappers.PostMapper
}

var logger = logging.GetLogger()

func NewPostService(repo PostRepository, fileStorage storage.FileStorage, cacheService caching.CacheService, producer messaging.Producer) *PostService {
	return &PostService{
		repo:         repo,
		fileStorage:  fileStorage,
		cacheService: cacheService,
		producer:     producer,
		mapper:       mappers.NewPostMapper(),
	}
}

func (ps *PostService) GetPosts(ctx context.Context) ([]response.PostResponse, error) {
	var postResponses []response.PostResponse
	if err := ps.getFromCache(ctx, "posts:list", &postResponses); err == nil {
		logger.Info("Fetched posts from cache")
		return postResponses, nil
	}

	posts, err := ps.repo.GetPosts(ctx)
	if err != nil {
		logger.Errorf("Error fetching posts: %v", err)
		return nil, err
	}

	postResponses = ps.mapper.MapEach(posts)
	_ = ps.setToCache(ctx, "posts:list", postResponses, 10*time.Minute)
	logger.Info("Fetched posts from DB and cached")

	return postResponses, nil
}

func (ps *PostService) GetPostByID(ctx context.Context, id uuid.UUID) (*response.PostResponse, error) {
	cacheKey := fmt.Sprintf("post:%s", id)
	var post response.PostResponse
	if err := ps.getFromCache(ctx, cacheKey, &post); err == nil {
		logger.Infof("Fetched post %s from cache", id)
		return &post, nil
	}

	modelPost, err := ps.repo.GetPostByID(ctx, id)
	if err != nil {
		logger.Errorf("Error fetching post %s: %v", id, err)
		return nil, err
	}

	post = ps.mapper.Map(*modelPost)
	_ = ps.setToCache(ctx, cacheKey, post, 10*time.Minute)
	logger.Infof("Fetched post %s from DB and cached", id)

	return &post, nil
}

func (ps *PostService) CreatePost(ctx context.Context, p request.PostRequest) error {
	files, err := ps.uploadAndCategorizeData(ctx, p.Files)
	if err != nil {
		return err
	}

	media, err := ps.uploadAndCategorizeData(ctx, p.Media)
	if err != nil {
		return err
	}

	post := &model.Post{
		Content: p.Content,
		UserID:  p.UserID,
		Media:   media,
		Files:   files,
	}

	if err := ps.repo.CreatePost(ctx, post); err != nil {
		logger.Errorf("Error creating post: %v", err)
		return err
	}

	return ps.publishEvent(ctx, "PostCreated", events.PostCreated{
		PostID: post.ID,
	})
}

func (ps *PostService) UpdatePost(ctx context.Context, postId uuid.UUID, p request.PostRequest) error {
	post, err := ps.validatePermission(ctx, p.UserID, postId)
	if err != nil {
		return err
	}

	mediaOldUrls := extractURLs(post.Media)
	filesOldUrls := extractURLs(post.Files)

	if len(p.Media) > 0 {
		post.Media, err = ps.uploadAndCategorizeData(ctx, p.Media)
		if err != nil {
			return err
		}
	} else {
		post.Media = nil
	}

	if len(p.Files) > 0 {
		post.Files, err = ps.uploadAndCategorizeData(ctx, p.Files)
		if err != nil {
			return err
		}
	} else {
		post.Files = nil
	}

	post.Content = p.Content

	if err := ps.repo.UpdatePost(ctx, post); err != nil {
		logger.Errorf("Error updating post %s: %v", postId, err)
		return err
	}

	// Invalidate cache for this post and the list
	_ = ps.cacheService.Delete(ctx, fmt.Sprintf("post:%s", postId))
	_ = ps.cacheService.Delete(ctx, "posts:list")

	return ps.publishEvent(ctx, "PostUpdated", events.PostUpdated{
		PostID:       post.ID,
		MediaNewURLs: extractURLs(post.Media),
		MediaOldURLs: mediaOldUrls,
		FilesNewURLs: extractURLs(post.Files),
		FilesOldURLs: filesOldUrls,
	})
}

func (ps *PostService) DeletePost(ctx context.Context, postId, userId uuid.UUID) error {
	post, err := ps.validatePermission(ctx, userId, postId)
	if err != nil {
		return err
	}

	if err := ps.repo.DeletePost(ctx, postId); err != nil {
		logger.Errorf("Error deleting post %s: %v", postId, err)
		return err
	}

	// Invalidate cache
	_ = ps.cacheService.Delete(ctx, fmt.Sprintf("post:%s", postId))

	return ps.publishEvent(ctx, "PostDeleted", events.PostDeleted{
		PostID:    post.ID,
		MediaURLs: extractURLs(post.Media),
		FilesURLs: extractURLs(post.Files),
	})
}

func (ps *PostService) uploadAndCategorizeData(ctx context.Context, files []*multipart.FileHeader) ([]model.File, error) {
	mediaMap := map[string][]string{}

	for _, fh := range files {
		file, err := fh.Open()
		if err != nil {
			logger.Errorf("Failed to open file: %v", err)
			continue
		}
		defer file.Close()

		url, err := ps.fileStorage.UploadFile(ctx, file, fh) // ctx-aware upload
		if err != nil {
			logger.Errorf("Error uploading file: %v", err)
			continue
		}

		mediaType := detectDataType(fh.Filename)
		mediaMap[mediaType] = append(mediaMap[mediaType], url)
	}

	var result []model.File
	for t, urls := range mediaMap {
		result = append(result, model.File{
			Type: t,
			URLs: urls,
		})
	}
	return result, nil
}

func detectDataType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return "image"
	case ".mp4", ".mov", ".avi", ".mkv":
		return "video"
	default:
		return "file"
	}
}

func (ps *PostService) validatePermission(ctx context.Context, userId, postId uuid.UUID) (*model.Post, error) {
	post, err := ps.repo.GetPostByID(ctx, postId)
	if err != nil {
		return nil, errors.New("post not found")
	}
	if post.UserID != userId {
		return nil, errors.New("user not allowed")
	}
	return post, nil
}

func (ps *PostService) getFromCache(ctx context.Context, key string, dest interface{}) error {
	data, err := ps.cacheService.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func (ps *PostService) setToCache(ctx context.Context, key string, value interface{}, duration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return ps.cacheService.Set(ctx, key, data, duration)
}

func (ps *PostService) publishEvent(ctx context.Context, eventType string, event interface{}) error {
	if err := ps.producer.Produce(ctx, eventType, event); err != nil {
		logger.Errorf("Error sending %s event: %v", eventType, err)
		return err
	}
	logger.Infof("%s event published successfully", eventType)
	return nil
}

// helper
func extractURLs(media []model.File) []string {
	var urls []string
	for _, m := range media {
		urls = append(urls, m.URLs...)
	}
	return urls
}
