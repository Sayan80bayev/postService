package response

type PostResponse struct {
	ID        uint             `json:"id"`
	Title     string           `json:"title"`
	Author    UserResponse     `json:"author"`
	Category  CategoryResponse `json:"category"`
	ImageURL  *string          `json:"image_url,omitempty"`
	LikeCount int64            `json:"like_count"`
}
