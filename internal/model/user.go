package model

type User struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	Username string `gorm:"unique;not null" json:"username"`
}
