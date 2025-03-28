package service

import "postService/internal/model"

type CommentRepository interface {
	Create(comm *model.Comment) error
	Update(comm *model.Comment) error
	Delete(id int) error
}

type CommentService struct {
	r CommentRepository
}

func NewCommentService(r CommentRepository) *CommentService {
	return &CommentService{r: r}
}
