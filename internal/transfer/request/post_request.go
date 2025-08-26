package request

import (
	"github.com/google/uuid"
	"mime/multipart"
)

type PostRequest struct {
	UserID  uuid.UUID `json:"user_id"` // UUID
	Content string    `json:"content" binding:"required" form:"content"`

	// Arrays of files for images and other files
	Media []*multipart.FileHeader
	Files []*multipart.FileHeader
}
