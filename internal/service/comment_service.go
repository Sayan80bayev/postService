package service

import (
	"errors"
	"postService/internal/model"
)

type CommentRepository interface {
	GetByID(id int) (*model.Comment, error)
	Create(comm *model.Comment) error
	Update(comm *model.Comment) error
	Delete(id int) error
}

type CommentService struct {
	r  CommentRepository
	pr PostRepository
}

func NewCommentService(r CommentRepository, pr PostRepository) *CommentService {
	return &CommentService{
		r:  r,
		pr: pr,
	}
}

// CreateComment creates a new comment after verifying the associated post exists
func (s *CommentService) CreateComment(comment *model.Comment) error {
	if comment == nil {
		return errors.New("comment is nil")
	}

	// Check if post exists
	_, err := s.pr.GetPostByID(int(comment.PostID))
	if err != nil {
		return errors.New("post not found")
	}

	return s.r.Create(comment)
}

// GetCommentByID retrieves a comment by ID
func (s *CommentService) GetCommentByID(id int) (*model.Comment, error) {
	if id <= 0 {
		return nil, errors.New("invalid comment ID")
	}
	return s.r.GetByID(id)
}

// UpdateComment updates an existing comment
func (s *CommentService) UpdateComment(comment *model.Comment) error {
	if comment == nil {
		return errors.New("comment is nil")
	}

	// Ensure comment exists
	existing, err := s.r.GetByID(int(comment.ID))
	if err != nil || existing == nil {
		return errors.New("comment not found")
	}

	// Optionally: validate post still exists
	_, err = s.pr.GetPostByID(int(comment.PostID))
	if err != nil {
		return errors.New("associated post not found")
	}

	return s.r.Update(comment)
}

// DeleteComment deletes a comment by ID
func (s *CommentService) DeleteComment(id int) error {
	if id <= 0 {
		return errors.New("invalid comment ID")
	}

	// Ensure comment exists
	_, err := s.r.GetByID(id)
	if err != nil {
		return errors.New("comment not found")
	}

	return s.r.Delete(id)
}
