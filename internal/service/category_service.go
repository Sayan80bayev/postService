package service

import (
	"postService/internal/models"
	"postService/internal/repository"
)

type CategoryService struct {
	categoryRepository *repository.CategoryRepository
}

func NewCategoryService(categoryRepository *repository.CategoryRepository) *CategoryService {
	return &CategoryService{categoryRepository: categoryRepository}
}

func (cs *CategoryService) CreateCategory(name string) error {
	category := &models.Category{
		Name: name,
	}
	return cs.categoryRepository.CreateCategory(category)
}

func (cs *CategoryService) GetCategories() ([]models.Category, error) {
	return cs.categoryRepository.GetCategories()
}

func (cs *CategoryService) DeleteCategory(id uint) error {
	return cs.categoryRepository.DeleteCategory(id)
}
