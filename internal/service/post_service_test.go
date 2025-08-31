package service

import (
	"context"
	"errors"
	"github.com/google/uuid"
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

func (m *MockPostRepo) CreatePost(ctx context.Context, post *model.Post) error {
	return m.Called(ctx, post).Error(0)
}

func (m *MockPostRepo) GetPosts(ctx context.Context) ([]model.Post, error) {
	args := m.Called(ctx)
	return args.Get(0).([]model.Post), args.Error(1)
}

func (m *MockPostRepo) GetPostByID(ctx context.Context, id uuid.UUID) (*model.Post, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.Post), args.Error(1)
}

func (m *MockPostRepo) UpdatePost(ctx context.Context, post *model.Post) error {
	return m.Called(ctx, post).Error(0)
}

func (m *MockPostRepo) DeletePost(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}, duration time.Duration) error {
	return m.Called(ctx, key, value, duration).Error(0)
}

func (m *MockStorage) UploadFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error) {
	args := m.Called(ctx, file, header)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) DeleteFileByURL(ctx context.Context, fileURL string) error {
	args := m.Called(ctx, fileURL)
	return args.Error(0)
}

func (m *MockProducer) Produce(ctx context.Context, eventType string, event interface{}) error {
	return m.Called(ctx, eventType, event).Error(0)
}

func (m *MockProducer) Close() {}

var (
	postID = uuid.New()
	userID = uuid.New()
)

// Tests

func TestPostService_CreatePost_Success(t *testing.T) {
	ctx := context.Background()

	repo := new(MockPostRepo)
	cache := new(MockCache)
	storage := new(MockStorage)
	producer := new(MockProducer)

	service := NewPostService(repo, storage, cache, producer)

	req := request.PostRequest{
		Content: "Hello world",
		UserID:  uuid.New(),
		Media:   []*multipart.FileHeader{},
	}

	repo.On("CreatePost", ctx, mock.Anything).Return(nil)
	producer.On("Produce", ctx, "PostCreated", mock.AnythingOfType("events.PostCreated")).Return(nil)

	err := service.CreatePost(ctx, req)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
	producer.AssertExpectations(t)
}

func TestPostService_GetPosts_CacheMiss_FetchFromRepo(t *testing.T) {
	ctx := context.Background()

	repo := new(MockPostRepo)
	cache := new(MockCache)
	storage := new(MockStorage)
	producer := new(MockProducer)

	service := NewPostService(repo, storage, cache, producer)

	cache.On("Get", ctx, "posts:list").Return("", errors.New("cache miss"))
	repo.On("GetPosts", ctx).Return([]model.Post{{ID: postID, Content: "Test", UserID: userID}}, nil)
	cache.On("Set", ctx, "posts:list", mock.Anything, mock.Anything).Return(nil)

	posts, err := service.GetPosts(ctx)
	assert.NoError(t, err)
	assert.Len(t, posts, 1)
}

func TestPostService_GetPostByID_CacheMiss(t *testing.T) {
	ctx := context.Background()

	repo := new(MockPostRepo)
	cache := new(MockCache)
	storage := new(MockStorage)
	producer := new(MockProducer)
	cacheKey := "post:" + postID.String()

	service := NewPostService(repo, storage, cache, producer)

	cache.On("Get", ctx, cacheKey).Return("", errors.New("cache miss"))
	repo.On("GetPostByID", ctx, postID).Return(&model.Post{ID: postID, UserID: userID}, nil)
	cache.On("Set", ctx, cacheKey, mock.Anything, mock.Anything).Return(nil)

	post, err := service.GetPostByID(ctx, postID)
	assert.NoError(t, err)
	assert.Equal(t, postID, post.ID)
}

func TestPostService_DeletePost_PermissionDenied(t *testing.T) {
	ctx := context.Background()

	repo := new(MockPostRepo)
	cache := new(MockCache)
	storage := new(MockStorage)
	producer := new(MockProducer)
	userID2 := uuid.New()

	service := NewPostService(repo, storage, cache, producer)

	repo.On("GetPostByID", ctx, postID).Return(&model.Post{ID: postID, UserID: userID}, nil)

	err := service.DeletePost(ctx, postID, userID2)
	assert.EqualError(t, err, "user not allowed")
}

func TestPostService_DeletePost_Success(t *testing.T) {
	ctx := context.Background()

	repo := new(MockPostRepo)
	cache := new(MockCache)
	storage := new(MockStorage)
	producer := new(MockProducer)

	service := NewPostService(repo, storage, cache, producer)

	post := &model.Post{
		ID:     postID,
		UserID: userID,
		Media: []model.File{
			{
				Type: "image",
				URLs: []string{"img1.jpg", "img2.jpg"},
			},
		},
		Files: []model.File{
			{
				Type: "pdf",
				URLs: []string{"doc1.pdf"},
			},
		},
	}

	repo.On("GetPostByID", ctx, postID).Return(post, nil)
	repo.On("DeletePost", ctx, postID).Return(nil)
	cache.On("Delete", ctx, mock.Anything).Return(nil)
	producer.On("Produce", ctx, "PostDeleted", events.PostDeleted{
		PostID:    postID,
		MediaURLs: []string{"img1.jpg", "img2.jpg"},
		FilesURLs: []string{"doc1.pdf"},
	}).Return(nil)

	err := service.DeletePost(ctx, postID, userID)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
	producer.AssertExpectations(t)
}
