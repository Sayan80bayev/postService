package events

import "github.com/google/uuid"

type PostCreated struct {
	PostID uuid.UUID `json:"post_id"`
}

type PostUpdated struct {
	PostID       uuid.UUID `json:"post_id"`
	MediaNewURLs []string  `json:"media_new_urls"`
	MediaOldURLs []string  `json:"media_old_urls"`
	FilesNewURLs []string  `json:"files_new_urls"`
	FilesOldURLs []string  `json:"files_old_urls"`
}

type PostDeleted struct {
	PostID    uuid.UUID `json:"post_id"`
	MediaURLs []string  `json:"media_urls"`
	FilesURLs []string  `json:"files_urls"`
}
