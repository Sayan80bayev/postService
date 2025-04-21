package model

import "gorm.io/gorm"

type Comment struct {
	gorm.Model
	ID      uint   `gorm:"primaryKey" json:"id"`
	PostID  int    `json:"post_id"`
	Post    Post   `gorm:"foreignKey:PostID"` // Связь с постом
	UserID  int    `json:"user_id"`
	User    Post   `gorm:"foreignKey:UserID"` // Связь с постом
	Content string `json:"content"`
}
