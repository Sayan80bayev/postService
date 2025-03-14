package repository

import (
	"gorm.io/gorm"
	"postService/internal/models"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db}
}

func (r *CategoryRepository) CreateCategory(category *models.Category) error {
	return r.db.Create(category).Error
}

func (r *CategoryRepository) DeleteCategory(id uint) error {
	return r.db.Delete(&models.Category{}, id).Error
}
func (r *CategoryRepository) UpdateCategory(category *models.Category) error {
	return r.db.Save(category).Error
}
func (r *CategoryRepository) GetCategoryById(id uint) (*models.Category, error) {
	var category models.Category
	err := r.db.First(&category, id).Error
	return &category, err
}
func (r *CategoryRepository) GetCategoryByName(name string) (*models.Category, error) {
	var category models.Category
	err := r.db.Where("name = ?", name).First(&category).Error
	return &category, err
}
func (r *CategoryRepository) GetCategories() ([]models.Category, error) {
	var categories []models.Category
	err := r.db.Find(&categories).Error
	return categories, err
}
