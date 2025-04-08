package service

import (
	"errors"
	"postService/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Мок-реализация CategoryRepository
type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) CreateCategory(category *model.Category) error {
	args := m.Called(category)
	return args.Error(0)
}

func (m *MockCategoryRepository) DeleteCategory(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockCategoryRepository) UpdateCategory(category *model.Category) error {
	args := m.Called(category)
	return args.Error(0)
}

func (m *MockCategoryRepository) GetCategoryById(id uint) (*model.Category, error) {
	args := m.Called(id)
	return args.Get(0).(*model.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetCategoryByName(name string) (*model.Category, error) {
	args := m.Called(name)
	return args.Get(0).(*model.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetCategories() ([]model.Category, error) {
	args := m.Called()
	return args.Get(0).([]model.Category), args.Error(1)
}

func TestCategoryService_CreateCategory(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	service := &CategoryService{categoryRepository: mockRepo}

	categoryName := "Tech"
	mockRepo.On("CreateCategory", mock.MatchedBy(func(c *model.Category) bool {
		return c.Name == categoryName
	})).Return(nil)

	err := service.CreateCategory(categoryName)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_CreateCategory_Error(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	service := &CategoryService{categoryRepository: mockRepo}

	categoryName := "Duplicate"
	expectedErr := errors.New("duplicate key")

	mockRepo.On("CreateCategory", mock.Anything).Return(expectedErr)

	err := service.CreateCategory(categoryName)
	assert.EqualError(t, err, expectedErr.Error())
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_GetCategories(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	service := &CategoryService{categoryRepository: mockRepo}

	categories := []model.Category{
		{Name: "Tech"},
		{Name: "Art"},
	}
	mockRepo.On("GetCategories").Return(categories, nil)

	result, err := service.GetCategories()
	assert.NoError(t, err)
	assert.Equal(t, categories, result)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_DeleteCategory(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	service := &CategoryService{categoryRepository: mockRepo}

	categoryID := uint(1)
	mockRepo.On("DeleteCategory", categoryID).Return(nil)

	err := service.DeleteCategory(categoryID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
