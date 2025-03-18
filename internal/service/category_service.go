package service

import (
	"postService/internal/model"
	"postService/internal/repository"
)

type CategoryRepository interface {
	CreateCategory(category *model.Category) error

	DeleteCategory(id uint) error

	UpdateCategory(category *model.Category) error

	GetCategoryById(id uint) (*model.Category, error)

	GetCategoryByName(name string) (*model.Category, error)

	GetCategories() ([]model.Category, error)
}

type CategoryService struct {
	categoryRepository CategoryRepository
}

func NewCategoryService(categoryRepository *repository.CategoryRepositoryImpl) *CategoryService {
	return &CategoryService{categoryRepository: categoryRepository}
}

func (cs *CategoryService) CreateCategory(name string) error {
	category := &model.Category{Name: name}
	return cs.categoryRepository.CreateCategory(category)
}

func (cs *CategoryService) GetCategories() ([]model.Category, error) {
	return cs.categoryRepository.GetCategories()
}

func (cs *CategoryService) DeleteCategory(id uint) error {
	return cs.categoryRepository.DeleteCategory(id)
}
