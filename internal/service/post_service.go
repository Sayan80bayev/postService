package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sayan80bayev/go-project/pkg/caching"
	"github.com/Sayan80bayev/go-project/pkg/logging"
	"github.com/Sayan80bayev/go-project/pkg/messaging"
	storage "github.com/Sayan80bayev/go-project/pkg/objectStorage"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"mime/multipart"
	"path/filepath"
	"postService/internal/cache"
	"postService/internal/events"
	"postService/internal/mappers"
	"postService/internal/model"
	"postService/internal/transfer/request"
	"postService/internal/transfer/response"
	"strings"
	"time"
)

// Cache keys and configuration constants
const (
	MediaTypeImage = "image"
	MediaTypeVideo = "video"
	MediaTypeFile  = "file"
)

type PostRepository interface {
	CreatePost(ctx context.Context, post *model.Post) error
	GetPosts(ctx context.Context, page, limit int64) (*model.PaginatedPosts, error)
	GetPostsByUserID(ctx context.Context, userID uuid.UUID, page, limit int64) (*model.PaginatedPosts, error)
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
	logger       *logrus.Logger
}

func NewPostService(repo PostRepository, fileStorage storage.FileStorage, cacheService caching.CacheService, producer messaging.Producer) *PostService {
	return &PostService{
		repo:         repo,
		fileStorage:  fileStorage,
		cacheService: cacheService,
		producer:     producer,
		mapper:       mappers.NewPostMapper(),
		logger:       logging.GetLogger(),
	}
}

// GetPosts with pagination
func (ps *PostService) GetPosts(ctx context.Context, page, limit int64) (response.PaginatedPostsResponse, error) {
	cacheKey := fmt.Sprintf("%s_page_%d_limit_%d", cache.CacheKeyPostsList, page, limit)

	var postResponses response.PaginatedPostsResponse
	if err := ps.getFromCache(ctx, cacheKey, &postResponses); err == nil {
		ps.logger.Infof("Fetched posts (page=%d, limit=%d) from cache", page, limit)
		return postResponses, nil
	}

	posts, err := ps.repo.GetPosts(ctx, page, limit)
	if err != nil {
		return response.PaginatedPostsResponse{}, fmt.Errorf("failed to fetch posts from DB: %w", err)
	}

	postResponses = ps.mapper.MapPaginated(*posts)
	if err := ps.setToCache(ctx, cacheKey, postResponses, cache.CacheTTL); err != nil {
		ps.logger.Warnf("failed to set posts (page=%d, limit=%d) to cache: %v", page, limit, err)
	}

	ps.logger.Infof("Fetched posts (page=%d, limit=%d) from DB and cached", page, limit)
	return postResponses, nil
}

// GetPostsByUserID with pagination
func (ps *PostService) GetPostsByUserID(ctx context.Context, userID uuid.UUID, page, limit int64) (response.PaginatedPostsResponse, error) {
	cacheKey := fmt.Sprintf("posts_user_%s_page_%d_limit_%d", userID, page, limit)

	var postResponses response.PaginatedPostsResponse
	if err := ps.getFromCache(ctx, cacheKey, &postResponses); err == nil {
		ps.logger.Infof("Fetched posts for user %s (page=%d, limit=%d) from cache", userID, page, limit)
		return postResponses, nil
	}

	posts, err := ps.repo.GetPostsByUserID(ctx, userID, page, limit)
	if err != nil {
		return response.PaginatedPostsResponse{}, fmt.Errorf("failed to fetch posts for user %s from DB: %w", userID, err)
	}

	postResponses = ps.mapper.MapPaginated(*posts)
	if err := ps.setToCache(ctx, cacheKey, postResponses, cache.CacheTTL); err != nil {
		ps.logger.Warnf("failed to set user posts (user=%s, page=%d, limit=%d) to cache: %v", userID, page, limit, err)
	}

	ps.logger.Infof("Fetched posts for user %s (page=%d, limit=%d) from DB and cached", userID, page, limit)
	return postResponses, nil
}

func (ps *PostService) GetPostByID(ctx context.Context, id uuid.UUID) (*response.PostResponse, error) {
	cacheKey := fmt.Sprintf(cache.CacheKeyPostFmt, id)
	var post response.PostResponse
	if err := ps.getFromCache(ctx, cacheKey, &post); err == nil {
		ps.logger.Infof("Fetched post %s from cache", id)
		return &post, nil
	}

	modelPost, err := ps.repo.GetPostByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch post %s from DB: %w", id, err)
	}

	post = ps.mapper.MapPost(*modelPost)
	if err := ps.setToCache(ctx, cacheKey, post, cache.CacheTTL); err != nil {
		ps.logger.Warnf("failed to set post %s to cache: %v", id, err)
	}

	ps.logger.Infof("Fetched post %s from DB and cached", id)
	return &post, nil
}

func (ps *PostService) CreatePost(ctx context.Context, p request.PostRequest) error {
	files, err := ps.uploadAndCategorizeData(ctx, p.Files)
	if err != nil {
		return fmt.Errorf("failed to upload files: %w", err)
	}

	media, err := ps.uploadAndCategorizeData(ctx, p.Media)
	if err != nil {
		return fmt.Errorf("failed to upload media: %w", err)
	}

	post := &model.Post{
		Content: p.Content,
		UserID:  p.UserID,
		Media:   media,
		Files:   files,
	}

	if err := ps.repo.CreatePost(ctx, post); err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	return ps.publishEvent(ctx, "PostCreatedEvent", events.PostCreatedEvent{
		PostID: post.ID,
	})
}

func (ps *PostService) UpdatePost(ctx context.Context, postId uuid.UUID, p request.PostRequest) error {
	post, err := ps.validatePermission(ctx, p.UserID, postId)
	if err != nil {
		return fmt.Errorf("permission validation failed for post %s: %w", postId, err)
	}

	mediaOldUrls := extractURLs(post.Media)
	filesOldUrls := extractURLs(post.Files)

	if len(p.Media) > 0 {
		post.Media, err = ps.uploadAndCategorizeData(ctx, p.Media)
		if err != nil {
			return fmt.Errorf("failed to upload media for post %s: %w", postId, err)
		}
	} else {
		post.Media = nil
	}

	if len(p.Files) > 0 {
		post.Files, err = ps.uploadAndCategorizeData(ctx, p.Files)
		if err != nil {
			return fmt.Errorf("failed to upload files for post %s: %w", postId, err)
		}
	} else {
		post.Files = nil
	}

	post.Content = p.Content

	if err := ps.repo.UpdatePost(ctx, post); err != nil {
		return fmt.Errorf("failed to update post %s: %w", postId, err)
	}

	if err := ps.cacheService.Delete(ctx, fmt.Sprintf(cache.CacheKeyPostFmt, postId)); err != nil {
		ps.logger.Warnf("failed to delete cache for post %s: %v", postId, err)
	}
	if err := ps.cacheService.Delete(ctx, cache.CacheKeyPostsList); err != nil {
		ps.logger.Warnf("failed to delete posts list cache: %v", err)
	}

	return ps.publishEvent(ctx, "PostUpdatedEvent", events.PostUpdatedEvent{
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
		return fmt.Errorf("permission validation failed for post %s: %w", postId, err)
	}

	if err := ps.repo.DeletePost(ctx, postId); err != nil {
		return fmt.Errorf("failed to delete post %s: %w", postId, err)
	}

	if err := ps.cacheService.Delete(ctx, fmt.Sprintf(cache.CacheKeyPostFmt, postId)); err != nil {
		ps.logger.Warnf("failed to delete cache for post %s: %v", postId, err)
	}

	return ps.publishEvent(ctx, "PostDeletedEvent", events.PostDeletedEvent{
		PostID:    post.ID,
		MediaURLs: extractURLs(post.Media),
		FilesURLs: extractURLs(post.Files),
	})
}

func (ps *PostService) uploadAndCategorizeData(ctx context.Context, files []*multipart.FileHeader) ([]model.File, error) {
	mediaMap := map[string][]string{}
	var allErrors []error

	for _, fh := range files {
		file, err := fh.Open()
		if err != nil {
			allErrors = append(allErrors, fmt.Errorf("failed to open file %s: %w", fh.Filename, err))
			continue
		}

		url, err := ps.fileStorage.UploadFile(ctx, file, fh)

		// always close the file and capture close error
		if cerr := file.Close(); cerr != nil {
			if err != nil {
				// if upload failed and close failed, combine information
				allErrors = append(allErrors, fmt.Errorf("failed to upload file %s: %w; additionally failed to close file: %v", fh.Filename, err, cerr))
				continue
			}
			// upload succeeded but close failed — report close error
			allErrors = append(allErrors, fmt.Errorf("failed to close file %s: %w", fh.Filename, cerr))
		}

		if err != nil {
			allErrors = append(allErrors, fmt.Errorf("failed to upload file %s: %w", fh.Filename, err))
			continue
		}

		mediaType := detectDataType(fh.Filename)
		mediaMap[mediaType] = append(mediaMap[mediaType], url)
	}

	if len(allErrors) > 0 {
		return nil, fmt.Errorf("encountered errors while uploading files: %v", allErrors)
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
		return MediaTypeImage
	case ".mp4", ".mov", ".avi", ".mkv":
		return MediaTypeVideo
	default:
		return MediaTypeFile
	}
}

func (ps *PostService) validatePermission(ctx context.Context, userId, postId uuid.UUID) (*model.Post, error) {
	post, err := ps.repo.GetPostByID(ctx, postId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch post %s: %w", postId, err)
	}
	if post.UserID != userId {
		return nil, errors.New("user not allowed to modify this post")
	}
	return post, nil
}

func (ps *PostService) getFromCache(ctx context.Context, key string, dest interface{}) error {
	data, err := ps.cacheService.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get key %s from cache: %w", key, err)
	}
	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal cache value for key %s: %w", key, err)
	}
	return nil
}

func (ps *PostService) setToCache(ctx context.Context, key string, value interface{}, duration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value for key %s: %w", key, err)
	}
	if err := ps.cacheService.Set(ctx, key, data, duration); err != nil {
		return fmt.Errorf("failed to set cache for key %s: %w", key, err)
	}
	return nil
}

func (ps *PostService) publishEvent(ctx context.Context, eventType string, event interface{}) error {
	if err := ps.producer.Produce(ctx, eventType, event); err != nil {
		return fmt.Errorf("failed to publish %s event: %w", eventType, err)
	}
	ps.logger.Infof("%s event published successfully", eventType)
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
