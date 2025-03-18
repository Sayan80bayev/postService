package repository

import (
	"fmt"
	"gorm.io/gorm"
	"postService/internal/model"
)

type PostRepositoryImpl struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) *PostRepositoryImpl {
	return &PostRepositoryImpl{db}
}

func (r *PostRepositoryImpl) CreatePost(post *model.Post) error {
	var category model.Category
	if err := r.db.First(&category, post.CategoryID).Error; err != nil {
		return fmt.Errorf("category with ID %d does not exist", post.CategoryID)
	}
	return r.db.Create(post).Error
}

func (r *PostRepositoryImpl) GetPosts() ([]model.Post, error) {
	var posts []model.Post
	err := r.db.Preload("User").Preload("Category").Find(&posts).Error
	return posts, err
}

func (r *PostRepositoryImpl) GetPostByID(id int) (*model.Post, error) {
	var post model.Post
	err := r.db.Preload("User").First(&post, id).Error
	return &post, err
}

func (r *PostRepositoryImpl) UpdatePost(post *model.Post) error {
	return r.db.Save(post).Error
}

func (r *PostRepositoryImpl) DeletePost(id int) error {
	return r.db.Delete(&model.Post{}, id).Error
}
