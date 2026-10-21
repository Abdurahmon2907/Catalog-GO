package main

import (
	"net/http"
	"os"
	"strings"

	"catalog-go/internal/database"
	"catalog-go/internal/models"

	"github.com/gin-gonic/gin"

	"fmt"
	"github.com/golang-jwt/jwt/v5"

	"golang.org/x/crypto/bcrypt"
	"strconv"
	"time"
)

var jwtSecret = []byte("secret_key")

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		isAdmin, exists := c.Get("is_admin")

		if !exists || isAdmin != true {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
			c.Abort()
			return
		}

		c.Next()
	}
}
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		tokenString := c.GetHeader("Authorization")

		if tokenString == "" {
			c.JSON(401, gin.H{"error": "no token"})
			c.Abort()
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if ok {
			fmt.Println("is_admin:", claims["is_admin"])
			c.Set("username", claims["username"])
			c.Set("is_admin", claims["is_admin"])
		}

		c.Next()
	}
}

func parseUint(s string) uint {
	n, _ := strconv.ParseUint(s, 10, 64)
	return uint(n)
}

func main() {
	r := gin.Default()

	var jwtSecret = []byte("secret_key")
	database.Connect()
	database.DB.AutoMigrate(&models.Post{})

	r.Static("/css", "./web/css")
	r.Static("/js", "./web/js")
	r.Static("/img", "./web/img")
	r.Static("/video", "./web/video")

	r.StaticFile("/", "./web/index.html")

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		filePath := "./web" + path

		if _, err := os.Stat(filePath); err == nil {
			c.File(filePath)
			return
		}

		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
	})

	r.GET("/api/posts", AuthMiddleware(), func(c *gin.Context) {
		var posts []models.Post
		database.DB.Find(&posts)

		c.JSON(200, posts)
	})
	r.POST("/api/posts", AuthMiddleware(), func(c *gin.Context) {
		var post models.Post

		if err := c.ShouldBindJSON(&post); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		username, _ := c.Get("username")
		post.Author = username.(string)

		database.DB.Create(&post)

		c.JSON(200, post)
	})
	r.PUT("/api/posts/:id", AuthMiddleware(), func(c *gin.Context) {
		id := c.Param("id")

		var post models.Post

		// найти пост
		if err := database.DB.First(&post, id).Error; err != nil {
			c.JSON(404, gin.H{"error": "post not found"})
			return
		}
		username, _ := c.Get("username")

		if post.Author != username.(string) {
			c.JSON(403, gin.H{"error": "you are not the owner"})
			return
		}

		// новые данные
		var input models.Post
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		post.Title = input.Title
		post.Content = input.Content
		post.Image = input.Image

		database.DB.Save(&post)

		c.JSON(200, post)
	})

	r.DELETE("/api/posts/:id", AuthMiddleware(), func(c *gin.Context) {

		id := c.Param("id")

		var post models.Post

		// ищем пост
		if err := database.DB.First(&post, id).Error; err != nil {
			c.JSON(404, gin.H{"error": "post not found"})
			return
		}

		// удаляем
		database.DB.Delete(&post)

		c.JSON(200, gin.H{"message": "post deleted"})
	})
	r.POST("/api/register", func(c *gin.Context) {
		var user models.User

		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if user.Username == "admin" {
			user.IsAdmin = true
		}

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		user.Password = string(hashedPassword)

		database.DB.Create(&user)

		c.JSON(http.StatusOK, gin.H{"message": "User created"})
	})

	r.POST("/api/login", func(c *gin.Context) {
		var input models.User
		var user models.User

		// получаем данные
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// ищем пользователя в БД
		if err := database.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		// проверяем пароль
		err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "wrong password"})
			return
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": user.Username,
			"is_admin": user.IsAdmin,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})
		tokenString, err := token.SignedString(jwtSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "login success",
			"token":   tokenString,
		})
	})

	r.POST("/api/posts/upload", AuthMiddleware(), func(c *gin.Context) {
		title := c.PostForm("title")
		content := c.PostForm("content")

		username, _ := c.Get("username")

		file, err := c.FormFile("image")

		imagePath := ""

		if err == nil {
			filename := strings.ReplaceAll(file.Filename, " ", "_")

			err = c.SaveUploadedFile(file, "./web/img/"+filename)
			if err != nil {
				c.JSON(500, gin.H{"error": "failed to save image"})
				return
			}

			imagePath = "/img/" + filename
		}

		post := models.Post{
			Title:   title,
			Content: content,
			Image:   imagePath,
			Author:  username.(string),
		}

		database.DB.Create(&post)

		c.JSON(200, post)
	})

	r.POST("/api/posts/:id/like", AuthMiddleware(), func(c *gin.Context) {
		id := c.Param("id")

		username, _ := c.Get("username")

		var post models.Post
		if err := database.DB.First(&post, id).Error; err != nil {
			c.JSON(404, gin.H{"error": "post not found"})
			return
		}

		var existingLike models.Like

		err := database.DB.
			Where("user = ? AND post_id = ?", username.(string), post.ID).
			First(&existingLike).Error

		if err == nil {
			database.DB.Delete(&existingLike)

			if post.Likes > 0 {
				post.Likes--
			}

			database.DB.Save(&post)

			c.JSON(200, gin.H{
				"message": "unliked",
				"likes":   post.Likes,
			})
			return
		}

		like := models.Like{
			User:   username.(string),
			PostID: post.ID,
		}

		database.DB.Create(&like)

		post.Likes++
		database.DB.Save(&post)

		c.JSON(200, gin.H{
			"message": "liked",
			"likes":   post.Likes,
		})
	})

	r.GET("/api/posts/:id/comments", AuthMiddleware(), func(c *gin.Context) {
		id := c.Param("id")

		var comments []models.Comment

		database.DB.Where("post_id = ?", id).Find(&comments)

		c.JSON(200, comments)
	})

	r.POST("/api/posts/:id/comments", AuthMiddleware(), func(c *gin.Context) {
		id := c.Param("id")
		username, _ := c.Get("username")

		var input models.Comment

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		comment := models.Comment{
			PostID:  parseUint(id),
			Author:  username.(string),
			Content: input.Content,
		}

		database.DB.Create(&comment)

		c.JSON(200, comment)
	})

	r.DELETE("/api/comments/:id", AuthMiddleware(), func(c *gin.Context) {
		id := c.Param("id")

		username, _ := c.Get("username")
		isAdmin, _ := c.Get("is_admin")

		var comment models.Comment

		if err := database.DB.First(&comment, id).Error; err != nil {
			c.JSON(404, gin.H{"error": "comment not found"})
			return
		}

		if !isAdmin.(bool) && comment.Author != username.(string) {
			c.JSON(403, gin.H{"error": "you are not the owner"})
			return
		}

		database.DB.Delete(&comment)

		c.JSON(200, gin.H{"message": "comment deleted"})
	})
	r.Run(":8080")

}
