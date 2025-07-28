package events

type PostCreated struct {
	PostID string `json:"post_id"`
}

type PostUpdated struct {
	PostID   string   `json:"post_id"`
	FileURLs []string `json:"file_urls"`
	OldURLs  []string `json:"old_urls"`
}

type PostDeleted struct {
	PostID    string   `json:"post_id"`
	ImageURLs []string `json:"image_urls"`
	FileURLs  []string `json:"file_urls"`
}
