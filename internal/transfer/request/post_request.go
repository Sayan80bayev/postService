package request

import "mime/multipart"

type PostRequest struct {
	Title      string `json:"title" binding:"required"`
	Content    string `json:"content" binding:"required"`
	CategoryID int    `json:"category_id" binding:"required"`
	UserID     int
	File       multipart.File
	Header     *multipart.FileHeader
}
