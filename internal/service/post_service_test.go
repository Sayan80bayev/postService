// Filename: service/post_service_test.go

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

type MockStorage struct{ mock.Mock }

type MockProducer struct{ mock.Mock }

func (m *MockPostRepo) CreatePost(post *model.Post) error {
	args := m.Called(post)
	return args.Error(0)
}

func (m *MockPostRepo) GetPosts() ([]model.Post, error) {
	args := m.Called()
	return args.Get(0).([]model.Post), args.Error(1)
}

func (m *MockPostRepo) GetPostByID(id int) (*model.Post, error) {
	args := m.Called(id)
	return args.Get(0).(*model.Post), args.Error(1)
}

func (m *MockPostRepo) UpdatePost(post *model.Post) error {
	args := m.Called(post)
	return args.Error(0)
}

func (m *MockPostRepo) DeletePost(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}, duration time.Duration) error {
	args := m.Called(ctx, key, value, duration)
	return args.Error(0)
}

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

func (m *MockStorage) UploadFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	args := m.Called(file, header)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) DeleteFileByURL(fileURL string) error {
	args := m.Called(fileURL)
	return args.Error(0)
}

func (m *MockProducer) Produce(eventType string, event interface{}) error {
	args := m.Called(eventType, event)
	return args.Error(0)
}

func (m *MockProducer) Close() {}

// Tests

func TestPostService_CreatePost_Success(t *testing.T) {
	repo := new(MockPostRepo)
	cache := new(MockCache)
	s := new(MockStorage)
	producer := new(MockProducer)

	service := NewPostService(repo, s, cache, producer)

	req := request.PostRequest{
		Title:      "Post 1",
		Content:    "Content 1",
		UserID:     1,
		CategoryID: 2,
		File:       nil,
		Header:     nil,
	}

	repo.On("CreatePost", mock.Anything).Return(nil)
	producer.On("Produce", "PostCreated", mock.AnythingOfType("events.PostCreated")).Return(nil)

	err := service.CreatePost(req)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	producer.AssertExpectations(t)
}

func TestPostService_GetPosts_FromRepo(t *testing.T) {
	repo := new(MockPostRepo)
	cache := new(MockCache)
	storage := new(MockStorage)
	producer := new(MockProducer)

	service := NewPostService(repo, storage, cache, producer)

	cache.On("Get", mock.Anything, "posts:list").Return("", errors.New("cache miss"))
	repo.On("GetPosts").Return([]model.Post{{Title: "Title"}}, nil)
	cache.On("Set", mock.Anything, "posts:list", mock.Anything, mock.Anything).Return(nil)

	posts, err := service.GetPosts()
	assert.NoError(t, err)
	assert.Len(t, posts, 1)
	repo.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestPostService_DeletePost_PermissionDenied(t *testing.T) {
	repo := new(MockPostRepo)
	cache := new(MockCache)
	storage := new(MockStorage)
	producer := new(MockProducer)

	service := NewPostService(repo, storage, cache, producer)

	repo.On("GetPostByID", 1).Return(&model.Post{ID: 1, User: model.User{ID: 2}}, nil)

	err := service.DeletePost(1, 999) // user 999 trying to delete post of user 2
	assert.EqualError(t, err, "user not allowed")
	repo.AssertExpectations(t)
}

func TestPostService_DeletePost_Success(t *testing.T) {
	repo := new(MockPostRepo)
	cache := new(MockCache)
	s := new(MockStorage)
	producer := new(MockProducer)

	service := NewPostService(repo, s, cache, producer)

	post := &model.Post{ID: 1, User: model.User{ID: 1}, ImageURL: "img.jpg"}

	repo.On("GetPostByID", 1).Return(post, nil)
	repo.On("DeletePost", 1).Return(nil)
	producer.On("Produce", "PostDeleted", events.PostDeleted{PostID: 1, ImageURL: "img.jpg"}).Return(nil)

	err := service.DeletePost(1, 1)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
	producer.AssertExpectations(t)
}
