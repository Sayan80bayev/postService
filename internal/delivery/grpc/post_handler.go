package grpc

import (
	"context"
	postpb "github.com/Sayan80bayev/go-project/pkg/proto/post"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"postService/internal/service"
	"postService/internal/transfer/response"
	"time"
)

type PostHandler struct {
	postpb.UnimplementedPostServiceServer
	ps *service.PostService
}

func NewPostHandler(ps *service.PostService) *PostHandler {
	return &PostHandler{ps: ps}
}

func (h *PostHandler) GetPost(ctx context.Context, req *postpb.GetPostRequest) (*postpb.GetPostResponse, error) {
	postId, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, err
	}

	res, err := h.ps.GetPostByID(ctx, postId)
	if err != nil {
		return nil, err
	}

	toPbFiles := func([]response.FilesResponse) []string {
		result := make([]string, 0, len(res.Files))
		for _, f := range res.Files {
			result = append(result, f.URLs...)
		}
		return result
	}

	media := toPbFiles(res.Media)
	files := toPbFiles(res.Media)
	createdAt, err := buildTimestamp(res.CreatedAt)
	if err != nil {
		return nil, err
	}

	updatedAt, err := buildTimestamp(res.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &postpb.GetPostResponse{
		Id:           res.ID.String(),
		UserId:       res.UserID.String(),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		Content:      res.Content,
		LikeCount:    res.LikeCount,
		RepostCount:  res.RepostCount,
		CommentCount: res.CommentCount,
		Media:        media,
		Files:        files,
	}, nil
}

// example string: "2025-11-24T10:15:30Z"
func buildTimestamp(ts string) (*timestamppb.Timestamp, error) {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return nil, err
	}
	return timestamppb.New(t), nil
}
