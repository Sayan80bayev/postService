package request

import "mime/multipart"

type PostRequest struct {
	Content string `json:"content" binding:"required" form:"content"`
	UserID  string `json:"user_id"` // UUID

	// Arrays of files for images and other files
	Media []*multipart.FileHeader
	Files []*multipart.FileHeader
}
