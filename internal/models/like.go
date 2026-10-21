package models

type Like struct {
	ID     uint   `gorm:"primaryKey"`
	User   string
	PostID uint
}