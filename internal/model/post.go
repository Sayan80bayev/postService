package model

import "github.com/google/uuid"

type File struct {
	Type string   `bson:"type" json:"type"` // "image" or "video"
	URLs []string `bson:"urls" json:"urls,omitempty"`
}

type Post struct {
	ID     uuid.UUID `bson:"_id" json:"id"`
	UserID uuid.UUID `bson:"user_id" json:"user_id"`

	Content string `bson:"content" json:"content"`

	Media []File `bson:"media" json:"media,omitempty"`
	Files []File `bson:"files" json:"files,omitempty"`

	LikeCount    int64 `bson:"like_count" json:"like_count"`
	RepostCount  int64 `bson:"repost_count" json:"repost_count"`
	CommentCount int64 `bson:"comment_count" json:"comment_count"`

	CreatedAt string `bson:"created_at" json:"created_at"`
	UpdatedAt string `bson:"updated_at" json:"updated_at"`
}

type PaginatedPosts struct {
	Posts   []Post `json:"posts"`
	Page    int64  `json:"page"`
	Limit   int64  `json:"limit"`
	Total   int64  `json:"total"`
	HasNext bool   `json:"has_next"`
}
