package events

import "mime/multipart"

type PostUpdated struct {
	ID     uint                  `json:"id"`
	PostID uint                  `json:"post_id"`
	File   multipart.File        `json:"file"`
	Header *multipart.FileHeader `json:"header"`
	oldURL string                `json:"old_url"`
}
