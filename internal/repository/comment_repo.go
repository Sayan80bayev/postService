package repository

import (
	"gorm.io/gorm"
	"postService/internal/model"
)

type CommentRepositoryImpl struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepositoryImpl {
	return &CommentRepositoryImpl{db: db}
}

func (r *CommentRepositoryImpl) GetByPostID(pid int) ([]model.Comment, error) {
	var comments []model.Comment
	err := r.db.Find(&comments).Where("post_id = ?", pid).Error
	return comments, err
}

func (r *CommentRepositoryImpl) GetByID(id int) (*model.Comment, error) {
	comm := &model.Comment{}
	err := r.db.First(comm, id).Error
	return comm, err
}

func (r *CommentRepositoryImpl) Create(comm *model.Comment) error {
	return r.db.Create(comm).Error
}

func (r *CommentRepositoryImpl) Update(comm *model.Comment) error {
	return r.db.Save(comm).Error
}

func (r *CommentRepositoryImpl) Delete(id int) error {
	return r.db.Delete(model.Comment{}, id).Error
}
