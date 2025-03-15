package service

import "postService/internal/models"

type CategoryService interface {
	CreateCategory(name string) error

	GetCategories() ([]models.Category, error)
	
	DeleteCategory(id uint) error
}
