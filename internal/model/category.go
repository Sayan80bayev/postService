package model

import "gorm.io/gorm"

type Category struct {
	gorm.Model
	Name  string `gorm:"unique;not null" json:"name"`
	Posts []Post `gorm:"foreignKey:CategoryID;constraint:OnDelete:CASCADE;" json:"-"` // Связь с постами
}
