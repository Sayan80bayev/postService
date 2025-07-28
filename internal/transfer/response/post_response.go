package response

type PostResponse struct {
	ID           string   `json:"id"`
	Content      string   `json:"content"`
	UserID       string   `json:"user_id"`
	ImageURLs    []string `json:"image_urls,omitempty"`
	FileURLs     []string `json:"file_urls,omitempty"`
	LikeCount    int64    `json:"like_count"`
	RepostCount  int64    `json:"repost_count"`
	CommentCount int64    `json:"comment_count"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}
