package models

import "gorm.io/gorm"

type Post struct {
	gorm.Model
	ID         uint     `gorm:"primaryKey" json:"id"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	CategoryID uint     `json:"category_id" gorm:"not null default -1"`
	Category   Category `gorm:"foreignKey:CategoryID"`                  // Связь с категорией
	UserID     uint     `json:"user_id"`                                // ID автора поста
	User       User     `gorm:"foreignKey:UserID" json:"user" gorm:"-"` // Prevent GORM from migrating User
	ImageURL   string   `json:"image_url"`
	LikeCount  int64    `json:"like_count" gorm:"default:0"`
}
