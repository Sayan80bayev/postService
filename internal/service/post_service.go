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
	GetPostByID(id int) (*model.Post, error)
	UpdatePost(post *model.Post) error
	DeletePost(id int) error
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

func (ps *PostService) GetPostByID(id int) (*response.PostResponse, error) {
	cacheKey := fmt.Sprintf("post:%d", id)
	var post response.PostResponse
	if err := ps.getFromCache(cacheKey, &post); err == nil {
		logger.Infof("Fetched post %d from cache", id)
		return &post, nil
	}

	modelPost, err := ps.repo.GetPostByID(id)
	if err != nil {
		logger.Errorf("Error fetching post %d: %v", id, err)
		return nil, err
	}

	post = ps.mapper.Map(*modelPost)
	_ = ps.setToCache(cacheKey, post, 10*time.Minute)
	logger.Infof("Fetched post %d from DB and cached", id)

	return &post, nil
}

func (ps *PostService) CreatePost(p request.PostRequest) error {
	imageURL, err := ps.uploadFile(p.File, p.Header)
	if err != nil {
		return err
	}

	post := &model.Post{
		Title:      p.Title,
		Content:    p.Content,
		UserID:     p.UserID,
		CategoryID: p.CategoryID,
		ImageURL:   imageURL,
		LikeCount:  0,
	}

	if err := ps.repo.CreatePost(post); err != nil {
		logger.Errorf("Error creating post: %v", err)
		return err
	}

	return ps.publishEvent("PostCreated", events.PostCreated{PostID: int(post.ID)})
}

func (ps *PostService) UpdatePost(postId int, p request.PostRequest) error {
	post, err := ps.validatePermission(p.UserID, postId)
	if err != nil {
		return err
	}

	oldImageURL := post.ImageURL
	if p.File != nil && p.Header != nil {
		if post.ImageURL, err = ps.uploadFile(p.File, p.Header); err != nil {
			return err
		}
	}

	post.Title, post.Content, post.CategoryID = p.Title, p.Content, p.CategoryID

	if err := ps.repo.UpdatePost(post); err != nil {
		logger.Errorf("Error updating post %d: %v", postId, err)
		return err
	}

	event := events.PostUpdated{PostID: postId, FileURL: post.ImageURL, OldURL: oldImageURL}
	return ps.publishEvent("PostUpdated", event)
}

func (ps *PostService) DeletePost(postId, userId int) error {
	post, err := ps.validatePermission(userId, postId)
	if err != nil {
		return err
	}

	if err := ps.repo.DeletePost(postId); err != nil {
		logger.Errorf("Error deleting post %d: %v", postId, err)
		return err
	}

	event := events.PostDeleted{PostID: postId, ImageURL: post.ImageURL}
	return ps.publishEvent("PostDeleted", event)
}

func (ps *PostService) validatePermission(userId, postId int) (*model.Post, error) {
	post, err := ps.repo.GetPostByID(postId)
	if err != nil {
		return nil, errors.New("post not found")
	}
	if post.User.ID != uint(userId) {
		return nil, errors.New("user not allowed")
	}
	return post, nil
}

func (ps *PostService) uploadFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	if file == nil || header == nil {
		return "", nil
	}
	url, err := ps.fileStorage.UploadFile(file, header)
	if err != nil {
		logger.Errorf("Error uploading file: %v", err)
	}
	return url, err
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
