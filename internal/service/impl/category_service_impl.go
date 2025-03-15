package impl

import (
	"postService/internal/models"
	"postService/internal/repository"
	"postService/internal/service"
)

type CategoryServiceImpl struct {
	categoryRepository *repository.CategoryRepository
}

func NewCategoryService(categoryRepository *repository.CategoryRepository) service.CategoryService {
	return &CategoryServiceImpl{categoryRepository: categoryRepository}
}

func (cs *CategoryServiceImpl) CreateCategory(name string) error {
	category := &models.Category{Name: name}
	return cs.categoryRepository.CreateCategory(category)
}

func (cs *CategoryServiceImpl) GetCategories() ([]models.Category, error) {
	return cs.categoryRepository.GetCategories()
}

func (cs *CategoryServiceImpl) DeleteCategory(id uint) error {
	return cs.categoryRepository.DeleteCategory(id)
}
