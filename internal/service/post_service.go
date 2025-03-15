package service

import (
	"mime/multipart"
	"postService/internal/response"
)

type PostService interface {
	CreatePost(title, content string, userID, categoryID uint, file multipart.File, header *multipart.FileHeader) error

	GetPosts() ([]response.PostResponse, error)

	GetPostByID(id uint) (*response.PostResponse, error)

	UpdatePost(content, title string, userId, postId, categoryID uint, file multipart.File, header *multipart.FileHeader) error

	DeletePost(postId, userId uint) error
}
