package model

import "gorm.io/gorm"

type Comment struct {
	gorm.Model
	ID      uint   `gorm:"primaryKey" json:"id"`
	PostID  uint   `json:"post_id"`
	Post    Post   `gorm:"foreignKey:PostID"` // Связь с постом
	Content string `json:"content"`
}
