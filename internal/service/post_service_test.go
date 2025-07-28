package service

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"mime/multipart"
	"postService/internal/events"
	"postService/internal/model"
	"postService/internal/transfer/request"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks

type MockPostRepo struct{ mock.Mock }

type MockCache struct{ mock.Mock }

func (m *MockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCache) Publish(ctx context.Context, channel, message string) error {
	args := m.Called(ctx, channel, message)
	return args.Error(0)
}

func (m *MockCache) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCache) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	args := m.Called(ctx, channel)
	return args.Get(0).(*redis.PubSub)
}

type MockStorage struct{ mock.Mock }

func (m *MockStorage) DeleteFileByURL(fileURL string) error {
	args := m.Called(fileURL)
	return args.Error(0)
}

type MockProducer struct{ mock.Mock }

func (m *MockPostRepo) CreatePost(post *model.Post) error {
	return m.Called(post).Error(0)
}

func (m *MockPostRepo) GetPosts() ([]model.Post, error) {
	args := m.Called()
	return args.Get(0).([]model.Post), args.Error(1)
}

func (m *MockPostRepo) GetPostByID(id string) (*model.Post, error) {
	args := m.Called(id)
	return args.Get(0).(*model.Post), args.Error(1)
}

func (m *MockPostRepo) UpdatePost(post *model.Post) error {
	return m.Called(post).Error(0)
}

func (m *MockPostRepo) DeletePost(id string) error {
	return m.Called(id).Error(0)
}

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}, duration time.Duration) error {
	return m.Called(ctx, key, value, duration).Error(0)
}

func (m *MockStorage) UploadFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	return m.Called(file, header).String(0), m.Called(file, header).Error(1)
}

func (m *MockProducer) Produce(eventType string, event interface{}) error {
	return m.Called(eventType, event).Error(0)
}

func (m *MockProducer) Close() {}

// Tests

func TestPostService_CreatePost_Success(t *testing.T) {
	repo := new(MockPostRepo)
	cache := new(MockCache)
	storage := new(MockStorage)
	producer := new(MockProducer)

	service := NewPostService(repo, storage, cache, producer)

	req := request.PostRequest{
		Content: "Hello world",
		UserID:  "user123",
		Images:  []*multipart.FileHeader{},
		Files:   []*multipart.FileHeader{},
	}

	repo.On("CreatePost", mock.Anything).Return(nil)
	producer.On("Produce", "PostCreated", mock.AnythingOfType("events.PostCreated")).Return(nil)

	err := service.CreatePost(req)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
	producer.AssertExpectations(t)
}

func TestPostService_GetPosts_CacheMiss_FetchFromRepo(t *testing.T) {
	repo := new(MockPostRepo)
	cache := new(MockCache)
	storage := new(MockStorage)
	producer := new(MockProducer)

	service := NewPostService(repo, storage, cache, producer)

	cache.On("Get", mock.Anything, "posts:list").Return("", errors.New("cache miss"))
	repo.On("GetPosts").Return([]model.Post{{ID: "1", Content: "Test", UserID: "u1"}}, nil)
	cache.On("Set", mock.Anything, "posts:list", mock.Anything, mock.Anything).Return(nil)

	posts, err := service.GetPosts()
	assert.NoError(t, err)
	assert.Len(t, posts, 1)
}

func TestPostService_GetPostByID_CacheMiss(t *testing.T) {
	repo := new(MockPostRepo)
	cache := new(MockCache)
	storage := new(MockStorage)
	producer := new(MockProducer)

	service := NewPostService(repo, storage, cache, producer)

	cache.On("Get", mock.Anything, "post:123").Return("", errors.New("cache miss"))
	repo.On("GetPostByID", "123").Return(&model.Post{ID: "123", UserID: "u1"}, nil)
	cache.On("Set", mock.Anything, "post:123", mock.Anything, mock.Anything).Return(nil)

	post, err := service.GetPostByID("123")
	assert.NoError(t, err)
	assert.Equal(t, "123", post.ID)
}

func TestPostService_DeletePost_PermissionDenied(t *testing.T) {
	repo := new(MockPostRepo)
	cache := new(MockCache)
	storage := new(MockStorage)
	producer := new(MockProducer)

	service := NewPostService(repo, storage, cache, producer)

	repo.On("GetPostByID", "1").Return(&model.Post{ID: "1", UserID: "someone_else"}, nil)

	err := service.DeletePost("1", "user123")
	assert.EqualError(t, err, "user not allowed")
}

func TestPostService_DeletePost_Success(t *testing.T) {
	repo := new(MockPostRepo)
	cache := new(MockCache)
	storage := new(MockStorage)
	producer := new(MockProducer)

	service := NewPostService(repo, storage, cache, producer)

	post := &model.Post{
		ID:        "1",
		UserID:    "user123",
		ImageURLs: []string{"img1.jpg"},
		FileURLs:  []string{"file1.pdf"},
	}

	repo.On("GetPostByID", "1").Return(post, nil)
	repo.On("DeletePost", "1").Return(nil)
	producer.On("Produce", "PostDeleted", events.PostDeleted{
		PostID:    "1",
		ImageURLs: post.ImageURLs,
		FileURLs:  post.FileURLs,
	}).Return(nil)

	err := service.DeletePost("1", "user123")
	assert.NoError(t, err)
}
