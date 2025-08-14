package response

type FilesResponse struct {
	Type string   `json:"type"` // "image", "video", "file"
	URLs []string `json:"urls,omitempty"`
}

type PostResponse struct {
	ID           string          `json:"id"`
	Content      string          `json:"content"`
	UserID       string          `json:"user_id"`
	Media        []FilesResponse `json:"media,omitempty"`
	Files        []FilesResponse `json:"files,omitempty"`
	LikeCount    int64           `json:"like_count"`
	RepostCount  int64           `json:"repost_count"`
	CommentCount int64           `json:"comment_count"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}
