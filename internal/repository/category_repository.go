package repository

import (
	"gorm.io/gorm"
	"postService/internal/model"
)

type CategoryRepositoryImpl struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepositoryImpl {
	return &CategoryRepositoryImpl{db}
}

func (r *CategoryRepositoryImpl) CreateCategory(category *model.Category) error {
	return r.db.Create(category).Error
}

func (r *CategoryRepositoryImpl) DeleteCategory(id uint) error {
	return r.db.Delete(&model.Category{}, id).Error
}

func (r *CategoryRepositoryImpl) UpdateCategory(category *model.Category) error {
	return r.db.Save(category).Error
}

func (r *CategoryRepositoryImpl) GetCategoryById(id uint) (*model.Category, error) {
	var category model.Category
	err := r.db.First(&category, id).Error
	return &category, err
}

func (r *CategoryRepositoryImpl) GetCategoryByName(name string) (*model.Category, error) {
	var category model.Category
	err := r.db.Where("name = ?", name).First(&category).Error
	return &category, err
}

func (r *CategoryRepositoryImpl) GetCategories() ([]model.Category, error) {
	var categories []model.Category
	err := r.db.Find(&categories).Error
	return categories, err
}
