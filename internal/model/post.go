package model

type Post struct {
	ID        string   `bson:"_id" json:"id"`
	Content   string   `bson:"content" json:"content"`
	UserID    string   `bson:"user_id" json:"user_id"`
	ImageURLs []string `bson:"image_urls" json:"image_urls,omitempty"`
	FileURLs  []string `bson:"file_urls" json:"file_urls,omitempty"`

	LikeCount    int64 `bson:"like_count" json:"like_count"`
	RepostCount  int64 `bson:"repost_count" json:"repost_count"`
	CommentCount int64 `bson:"comment_count" json:"comment_count"`

	CreatedAt string `bson:"created_at" json:"created_at"`
	UpdatedAt string `bson:"updated_at" json:"updated_at"`
}
