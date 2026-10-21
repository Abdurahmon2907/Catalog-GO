package database

import (
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"catalog-go/internal/models"
)
var DB *gorm.DB

func Connect() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("DB ERROR:", err)
	}

	DB = db

	// 👇 ДОБАВЬ ЭТО
	err = db.AutoMigrate(&models.Post{}, &models.User{}, &models.Like{}, &models.Comment{})
	if err != nil {
		log.Fatal("MIGRATE ERROR:", err)
	}
}