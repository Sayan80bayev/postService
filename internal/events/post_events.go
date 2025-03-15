package events

type PostCreated struct {
	PostID uint `json:"post_id"`
}

type PostUpdated struct {
	PostID  uint   `json:"post_id"`
	FileURL string `json:"file_url"`
	OldURL  string `json:"old_url"`
}

type PostDeleted struct {
	PostID   uint   `json:"post_id"`
	ImageURL string `json:"image_url"`
}
