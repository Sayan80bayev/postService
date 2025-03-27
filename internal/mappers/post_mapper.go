package mappers

import (
	"postService/internal/model"
	response2 "postService/internal/transfer/response"
)

type PostMapper struct {
	MapFunc[model.Post, response2.PostResponse]
}

func NewPostMapper() *PostMapper {
	return &PostMapper{MapFunc: MapPostToResponse}
}

func MapPostToResponse(post model.Post) response2.PostResponse {
	return response2.PostResponse{
		ID:    post.ID,
		Title: post.Title,
		Author: response2.UserResponse{
			ID:       post.User.ID,
			Username: post.User.Username,
		},
		Category: response2.CategoryResponse{
			ID:   post.Category.ID,
			Name: post.Category.Name,
		},
		ImageURL:  &post.ImageURL,
		LikeCount: post.LikeCount,
	}
}
