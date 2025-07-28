package mappers

import (
	"postService/internal/model"
	"postService/internal/transfer/response"
)

type PostMapper struct {
	MapFunc[model.Post, response.PostResponse]
}

func NewPostMapper() *PostMapper {
	return &PostMapper{MapFunc: MapPostToResponse}
}

func MapPostToResponse(post model.Post) response.PostResponse {
	return response.PostResponse{
		ID:           post.ID,
		Content:      post.Content,
		UserID:       post.UserID,
		ImageURLs:    post.ImageURLs,
		FileURLs:     post.FileURLs,
		LikeCount:    post.LikeCount,
		RepostCount:  post.RepostCount,
		CommentCount: post.CommentCount,
		CreatedAt:    post.CreatedAt,
		UpdatedAt:    post.UpdatedAt,
	}
}
