package impl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"postService/internal/events"
	"postService/internal/mappers"
	"postService/internal/messaging"
	"postService/internal/models"
	"postService/internal/repository"
	"postService/internal/response"
	"postService/internal/service"
	"postService/pkg/logging"
	"time"
)

type PostServiceImpl struct {
	postRepo     *repository.PostRepository
	storage      service.FileStorage
	cacheService service.CacheService
	producer     *messaging.Producer
	mapper       *mappers.PostMapper
}

var logger = logging.GetLogger()

func NewPostService(
	postRepo *repository.PostRepository,
	storage service.FileStorage,
	cacheService service.CacheService,
	producer *messaging.Producer) service.PostService {

	return &PostServiceImpl{
		postRepo:     postRepo,
		storage:      storage,
		cacheService: cacheService,
		producer:     producer,
		mapper:       mappers.NewPostMapper(),
	}
}

func (ps *PostServiceImpl) CreatePost(
	title, content string,
	userID, categoryID uint,
	file multipart.File,
	header *multipart.FileHeader) error {

	var imageURL string
	if file != nil && header != nil {
		uploadedURL, err := ps.storage.UploadFile(file, header)
		if err != nil {
			logger.Errorf("Error uploading file: %v", err)
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
		logger.Errorf("Error creating post: %v", err)
		return err
	}

	ps.producer.Produce("PostCreated", events.PostCreated{PostID: post.ID})
	logger.Infof("Post created successfully: %d", post.ID)
	return nil
}

func (ps *PostServiceImpl) GetPosts() ([]response.PostResponse, error) {
	ctx := context.TODO()
	cachedPosts, err := ps.cacheService.Get(ctx, "posts:list")
	if err == nil {
		var postResponses []response.PostResponse
		json.Unmarshal([]byte(cachedPosts), &postResponses)
		logger.Info("Fetched posts from cache")
		return postResponses, nil
	}

	posts, err := ps.postRepo.GetPosts()
	if err != nil {
		logger.Errorf("Error fetching posts: %v", err)
		return nil, err
	}

	postResponses := ps.mapper.MapEach(posts)
	jsonData, _ := json.Marshal(postResponses)
	ps.cacheService.Set(ctx, "posts:list", jsonData, 10*time.Minute)
	logger.Info("Fetched posts from DB and cached")

	return postResponses, nil
}

func (ps *PostServiceImpl) GetPostByID(id uint) (*response.PostResponse, error) {
	ctx := context.TODO()
	cacheKey := fmt.Sprintf("post:%d", id)

	cachedPost, err := ps.cacheService.Get(ctx, cacheKey)
	if err == nil {
		var post response.PostResponse
		json.Unmarshal([]byte(cachedPost), &post)
		logger.Infof("Fetched post %d from cache", id)
		return &post, nil
	}

	post, err := ps.postRepo.GetPostByID(id)
	if err != nil {
		logger.Errorf("Error fetching post %d: %v", id, err)
		return nil, err
	}

	resp := ps.mapper.Map(*post)
	jsonData, _ := json.Marshal(resp)
	ps.cacheService.Set(ctx, cacheKey, jsonData, 10*time.Minute)
	logger.Infof("Fetched post %d from DB and cached", id)

	return &resp, nil
}

func (ps *PostServiceImpl) UpdatePost(content, title string, userId, postId, categoryID uint, file multipart.File, header *multipart.FileHeader) error {
	post, err := validatePermission(userId, postId, ps)
	if err != nil {
		logger.Errorf("Permission denied for user %d on post %d", userId, postId)
		return err
	}

	oldImageURL := post.ImageURL
	if file != nil && header != nil {
		imageURL, err := ps.storage.UploadFile(file, header)
		if err != nil {
			logger.Errorf("Error uploading file for post %d: %v", postId, err)
			return err
		}
		post.ImageURL = imageURL
	}

	post.Content = content
	post.Title = title
	post.CategoryID = categoryID

	err = ps.postRepo.UpdatePost(post)
	if err != nil {
		logger.Errorf("Error updating post %d: %v", postId, err)
		return err
	}

	event := events.PostUpdated{
		PostID:  post.ID,
		FileURL: post.ImageURL,
		OldURL:  oldImageURL,
	}
	err = ps.producer.Produce("PostUpdated", event)
	if err != nil {
		logger.Errorf("Error sending PostUpdated event for post %d: %v", postId, err)
	}

	logger.Infof("Post %d updated successfully", postId)
	return nil
}

func (ps *PostServiceImpl) DeletePost(postId, userId uint) error {
	post, err := validatePermission(userId, postId, ps)
	if err != nil {
		logger.Errorf("Permission denied for user %d on post %d", userId, postId)
		return err
	}

	err = ps.postRepo.DeletePost(postId)
	if err != nil {
		logger.Errorf("Error deleting post %d: %v", postId, err)
		return err
	}

	event := events.PostDeleted{
		PostID:   postId,
		ImageURL: post.ImageURL,
	}
	err = ps.producer.Produce("PostDeleted", event)
	if err != nil {
		logger.Errorf("Error sending PostDeleted event for post %d: %v", postId, err)
	}

	logger.Infof("Post %d deleted successfully", postId)
	return nil
}

func validatePermission(userId, postId uint, ps *PostServiceImpl) (*models.Post, error) {
	post, err := ps.postRepo.GetPostByID(postId)
	if err != nil {
		logger.Errorf("Post %d not found", postId)
		return nil, errors.New("post not found")
	}
	if post.User.ID != userId {
		logger.Errorf("User %d is not allowed to modify post %d", userId, postId)
		return nil, errors.New("user not allowed")
	}
	return post, nil
}
