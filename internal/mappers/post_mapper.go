package mappers

import (
	"github.com/Sayan80bayev/go-project/pkg/mapper"
	"postService/internal/model"
	"postService/internal/transfer/response"
)

type PostMapper struct {
	MapPost      mapper.MapFunc[model.Post, response.PostResponse]
	MapPaginated mapper.MapFunc[model.PaginatedPosts, response.PaginatedPostsResponse]
}

func NewPostMapper() *PostMapper {
	return &PostMapper{
		MapPost:      MapPostToResponse,
		MapPaginated: MapPaginatedPostsToResponse,
	}
}

func MapPostToResponse(post model.Post) response.PostResponse {
	mediaResponses := make([]response.FilesResponse, len(post.Media))
	for i, m := range post.Media {
		mediaResponses[i] = response.FilesResponse{
			Type: m.Type,
			URLs: m.URLs,
		}
	}

	filesResponses := make([]response.FilesResponse, len(post.Files))
	for i, m := range post.Files {
		filesResponses[i] = response.FilesResponse{
			Type: m.Type,
			URLs: m.URLs,
		}
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

func MapPaginatedPostsToResponse(posts model.PaginatedPosts) response.PaginatedPostsResponse {
	postResponses := make([]response.PostResponse, len(posts.Posts))
	for i, p := range posts.Posts {
		postResponses[i] = MapPostToResponse(p)
	}

	return response.PaginatedPostsResponse{
		Posts:   postResponses,
		Page:    posts.Page,
		Limit:   posts.Limit,
		Total:   posts.Total,
		HasNext: posts.HasNext,
	}
}
