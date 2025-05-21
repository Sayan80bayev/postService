package request

import "mime/multipart"

type PostRequest struct {
	Title      string `json:"title" binding:"required" form:"title"`
	Content    string `json:"content" binding:"required" form:"content"`
	CategoryID int    `json:"category_id" binding:"required" form:"category_id"`
	UserID     int
	File       multipart.File
	Header     *multipart.FileHeader
}
