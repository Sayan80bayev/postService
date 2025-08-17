package mappers

import (
	"github.com/Sayan80bayev/go-project/pkg/mapper"
	"postService/internal/model"
	"postService/internal/transfer/response"
)

type PostMapper struct {
	mapper.MapFunc[model.Post, response.PostResponse]
}

func NewPostMapper() *PostMapper {
	return &PostMapper{MapFunc: MapPostToResponse}
}

func MapPostToResponse(post model.Post) response.PostResponse {
	var mediaResponses []response.FilesResponse
	for _, m := range post.Media {
		mediaResponses = append(mediaResponses, response.FilesResponse{
			Type: m.Type,
			URLs: m.URLs,
		})
	}

	var filesResponses []response.FilesResponse
	for _, m := range post.Files {
		filesResponses = append(filesResponses, response.FilesResponse{
			Type: m.Type,
			URLs: m.URLs,
		})
	}

	return response.PostResponse{
		ID:           post.ID,
		Content:      post.Content,
		UserID:       post.UserID,
		Media:        mediaResponses,
		Files:        filesResponses,
		LikeCount:    post.LikeCount,
		RepostCount:  post.RepostCount,
		CommentCount: post.CommentCount,
		CreatedAt:    post.CreatedAt,
		UpdatedAt:    post.UpdatedAt,
	}
}
