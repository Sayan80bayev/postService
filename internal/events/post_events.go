package events

type PostUpdated struct {
	PostID  uint   `json:"post_id"`
	FileURL string `json:"file_url"`
	OldURL  string `json:"old_url"`
}

type PostDeleted struct {
	PostID   uint   `json:"post_id"`
	ImageURL string `json:"image_url"`
}
