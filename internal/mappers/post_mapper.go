package mappers

import (
	"postService/internal/model"
	"postService/internal/response"
)

type PostMapper struct {
	MapFunc[model.Post, response.PostResponse]
}

func NewPostMapper() *PostMapper {
	return &PostMapper{MapFunc: MapPostToResponse}
}

func MapPostToResponse(post model.Post) response.PostResponse {
	return response.PostResponse{
		ID:    post.ID,
		Title: post.Title,
		Author: response.UserResponse{
			ID:       post.User.ID,
			Username: post.User.Username,
		},
		Category: response.CategoryResponse{
			ID:   post.Category.ID,
			Name: post.Category.Name,
		},
		ImageURL:  &post.ImageURL,
		LikeCount: post.LikeCount,
	}
}
