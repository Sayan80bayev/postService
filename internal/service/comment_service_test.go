package service_test

import (
	"errors"
	"postService/internal/model"
	"postService/internal/service"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCommentRepo mocks the CommentRepository interface
type MockCommentRepo struct {
	mock.Mock
}

func (m *MockCommentRepo) GetByID(id int) (*model.Comment, error) {
	args := m.Called(id)
	return args.Get(0).(*model.Comment), args.Error(1)
}

func (m *MockCommentRepo) Create(comm *model.Comment) error {
	args := m.Called(comm)
	return args.Error(0)
}

func (m *MockCommentRepo) Update(comm *model.Comment) error {
	args := m.Called(comm)
	return args.Error(0)
}

func (m *MockCommentRepo) Delete(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

// MockPostRepo mocks the PostRepository interface
type MockPostRepo struct {
	mock.Mock
}

func (m *MockPostRepo) GetPostByID(id int) (*model.Post, error) {
	args := m.Called(id)
	return args.Get(0).(*model.Post), args.Error(1)
}

func (m *MockPostRepo) CreatePost(post *model.Post) error { return nil }
func (m *MockPostRepo) GetPosts() ([]model.Post, error)   { return nil, nil }
func (m *MockPostRepo) UpdatePost(post *model.Post) error { return nil }
func (m *MockPostRepo) DeletePost(id int) error           { return nil }

func TestCreateComment_Success(t *testing.T) {
	mockCommentRepo := new(MockCommentRepo)
	mockPostRepo := new(MockPostRepo)
	s := service.NewCommentService(mockCommentRepo, mockPostRepo)

	comment := &model.Comment{
		ID:      1,
		PostID:  1,
		Content: "Nice post!",
	}

	mockPostRepo.On("GetPostByID", int(comment.PostID)).Return(&model.Post{}, nil)
	mockCommentRepo.On("Create", comment).Return(nil)

	err := s.CreateComment(comment)

	assert.NoError(t, err)
	mockPostRepo.AssertExpectations(t)
	mockCommentRepo.AssertExpectations(t)
}

func TestCreateComment_PostNotFound(t *testing.T) {
	mockCommentRepo := new(MockCommentRepo)
	mockPostRepo := new(MockPostRepo)
	s := service.NewCommentService(mockCommentRepo, mockPostRepo)

	comment := &model.Comment{
		PostID:  999,
		Content: "Comment with invalid post",
	}

	// Explicitly return nil of the correct type
	mockPostRepo.On("GetPostByID", int(comment.PostID)).Return((*model.Post)(nil), errors.New("not found"))

	err := s.CreateComment(comment)

	assert.EqualError(t, err, "post not found")
}

func TestGetCommentByID_Success(t *testing.T) {
	mockCommentRepo := new(MockCommentRepo)
	mockPostRepo := new(MockPostRepo)
	s := service.NewCommentService(mockCommentRepo, mockPostRepo)

	comment := &model.Comment{ID: 1, Content: "Test"}
	mockCommentRepo.On("GetByID", 1).Return(comment, nil)

	result, err := s.GetCommentByID(1)

	assert.NoError(t, err)
	assert.Equal(t, comment, result)
}

func TestGetCommentByID_InvalidID(t *testing.T) {
	s := service.NewCommentService(nil, nil)

	result, err := s.GetCommentByID(0)

	assert.Nil(t, result)
	assert.EqualError(t, err, "invalid comment ID")
}

func TestUpdateComment_Success(t *testing.T) {
	mockCommentRepo := new(MockCommentRepo)
	mockPostRepo := new(MockPostRepo)
	s := service.NewCommentService(mockCommentRepo, mockPostRepo)

	comment := &model.Comment{ID: 1, PostID: 1, Content: "Updated!"}
	mockCommentRepo.On("GetByID", 1).Return(comment, nil)
	mockPostRepo.On("GetPostByID", int(comment.PostID)).Return(&model.Post{}, nil)
	mockCommentRepo.On("Update", comment).Return(nil)

	err := s.UpdateComment(comment)

	assert.NoError(t, err)
}

func TestUpdateComment_CommentNotFound(t *testing.T) {
	mockCommentRepo := new(MockCommentRepo)
	mockPostRepo := new(MockPostRepo)
	s := service.NewCommentService(mockCommentRepo, mockPostRepo)

	comment := &model.Comment{ID: 1, PostID: 1}

	// Explicitly return nil of the correct type for comment not found
	mockCommentRepo.On("GetByID", 1).Return((*model.Comment)(nil), errors.New("not found"))

	err := s.UpdateComment(comment)

	assert.EqualError(t, err, "comment not found")
}

func TestDeleteComment_Success(t *testing.T) {
	mockCommentRepo := new(MockCommentRepo)
	mockPostRepo := new(MockPostRepo)
	s := service.NewCommentService(mockCommentRepo, mockPostRepo)

	comment := &model.Comment{ID: 1}
	mockCommentRepo.On("GetByID", 1).Return(comment, nil)
	mockCommentRepo.On("Delete", 1).Return(nil)

	err := s.DeleteComment(1)

	assert.NoError(t, err)
}

func TestDeleteComment_CommentNotFound(t *testing.T) {
	mockCommentRepo := new(MockCommentRepo)
	mockPostRepo := new(MockPostRepo)
	s := service.NewCommentService(mockCommentRepo, mockPostRepo)

	mockCommentRepo.On("GetByID", 1).Return((*model.Comment)(nil), errors.New("not found"))
	err := s.DeleteComment(1)

	assert.EqualError(t, err, "comment not found")
}
