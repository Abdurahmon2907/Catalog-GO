package models

type Comment struct {
	ID      uint   `gorm:"primaryKey"`
	PostID  uint
	Author  string
	Content string
}