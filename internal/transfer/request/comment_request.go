package request

type CommentRequest struct {
	PostID  int    `json:"post_id"`
	Content string `json:"content"`
}
