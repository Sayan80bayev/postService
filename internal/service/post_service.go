package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"postService/internal/events"
	"postService/internal/mappers"
	"postService/internal/messaging"
	"postService/internal/model"
	"postService/internal/pkg/storage"
	"postService/internal/transfer/request"
	"postService/internal/transfer/response"
	"postService/pkg/logging"
	"time"
)

type PostRepository interface {
	CreatePost(post *model.Post) error
	GetPosts() ([]model.Post, error)
	GetPostByID(id string) (*model.Post, error)
	UpdatePost(post *model.Post) error
	DeletePost(id string) error
}

type PostService struct {
	repo         PostRepository
	cacheService CacheService
	fileStorage  storage.FileStorage
	producer     messaging.Producer
	mapper       *mappers.PostMapper
}

var logger = logging.GetLogger()

func NewPostService(repo PostRepository, fileStorage storage.FileStorage, cacheService CacheService, producer messaging.Producer) *PostService {
	return &PostService{
		repo:         repo,
		fileStorage:  fileStorage,
		cacheService: cacheService,
		producer:     producer,
		mapper:       mappers.NewPostMapper(),
	}
}

func (ps *PostService) GetPosts() ([]response.PostResponse, error) {
	var postResponses []response.PostResponse
	if err := ps.getFromCache("posts:list", &postResponses); err == nil {
		logger.Info("Fetched posts from cache")
		return postResponses, nil
	}

	posts, err := ps.repo.GetPosts()
	if err != nil {
		logger.Errorf("Error fetching posts: %v", err)
		return nil, err
	}

	postResponses = ps.mapper.MapEach(posts)
	_ = ps.setToCache("posts:list", postResponses, 10*time.Minute)
	logger.Info("Fetched posts from DB and cached")

	return postResponses, nil
}

func (ps *PostService) GetPostByID(id string) (*response.PostResponse, error) {
	cacheKey := fmt.Sprintf("post:%s", id)
	var post response.PostResponse
	if err := ps.getFromCache(cacheKey, &post); err == nil {
		logger.Infof("Fetched post %s from cache", id)
		return &post, nil
	}

	modelPost, err := ps.repo.GetPostByID(id)
	if err != nil {
		logger.Errorf("Error fetching post %s: %v", id, err)
		return nil, err
	}

	post = ps.mapper.Map(*modelPost)
	_ = ps.setToCache(cacheKey, post, 10*time.Minute)
	logger.Infof("Fetched post %s from DB and cached", id)

	return &post, nil
}

func (ps *PostService) CreatePost(p request.PostRequest) error {
	imageURLs, err := ps.uploadFiles(p.Images)
	if err != nil {
		return err
	}
	fileURLs, err := ps.uploadFiles(p.Files)
	if err != nil {
		return err
	}

	post := &model.Post{
		Content:   p.Content,
		UserID:    p.UserID,
		ImageURLs: imageURLs,
		FileURLs:  fileURLs,
	}

	if err := ps.repo.CreatePost(post); err != nil {
		logger.Errorf("Error creating post: %v", err)
		return err
	}

	return ps.publishEvent("PostCreated", events.PostCreated{PostID: post.ID})
}

func (ps *PostService) UpdatePost(postId string, p request.PostRequest) error {
	post, err := ps.validatePermission(p.UserID, postId)
	if err != nil {
		return err
	}

	oldImageURLs := post.ImageURLs
	oldFileURLs := post.FileURLs

	if len(p.Images) > 0 {
		post.ImageURLs, err = ps.uploadFiles(p.Images)
		if err != nil {
			return err
		}
	}
	if len(p.Files) > 0 {
		post.FileURLs, err = ps.uploadFiles(p.Files)
		if err != nil {
			return err
		}
	}

	post.Content = p.Content

	if err := ps.repo.UpdatePost(post); err != nil {
		logger.Errorf("Error updating post %s: %v", postId, err)
		return err
	}

	event := events.PostUpdated{
		PostID:   post.ID,
		FileURLs: append(post.ImageURLs, post.FileURLs...),
		OldURLs:  append(oldImageURLs, oldFileURLs...),
	}
	return ps.publishEvent("PostUpdated", event)
}

func (ps *PostService) DeletePost(postId, userId string) error {
	post, err := ps.validatePermission(userId, postId)
	if err != nil {
		return err
	}

	if err := ps.repo.DeletePost(postId); err != nil {
		logger.Errorf("Error deleting post %s: %v", postId, err)
		return err
	}

	event := events.PostDeleted{
		PostID:    post.ID,
		ImageURLs: post.ImageURLs,
		FileURLs:  post.FileURLs,
	}
	return ps.publishEvent("PostDeleted", event)
}

func (ps *PostService) validatePermission(userId, postId string) (*model.Post, error) {
	post, err := ps.repo.GetPostByID(postId)
	if err != nil {
		return nil, errors.New("post not found")
	}
	if post.UserID != userId {
		return nil, errors.New("user not allowed")
	}
	return post, nil
}

func (ps *PostService) uploadFiles(files []*multipart.FileHeader) ([]string, error) {
	var urls []string
	for _, fh := range files {
		file, err := fh.Open()
		if err != nil {
			logger.Errorf("Failed to open file: %v", err)
			continue
		}
		defer file.Close()

		url, err := ps.fileStorage.UploadFile(file, fh)
		if err != nil {
			logger.Errorf("Error uploading file: %v", err)
			continue
		}
		urls = append(urls, url)
	}
	return urls, nil
}

func (ps *PostService) getFromCache(key string, dest interface{}) error {
	data, err := ps.cacheService.Get(context.TODO(), key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func (ps *PostService) setToCache(key string, value interface{}, duration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return ps.cacheService.Set(context.TODO(), key, data, duration)
}

func (ps *PostService) publishEvent(eventType string, event interface{}) error {
	if err := ps.producer.Produce(eventType, event); err != nil {
		logger.Errorf("Error sending %s event: %v", eventType, err)
		return err
	}
	logger.Infof("%s event published successfully", eventType)
	return nil
}
