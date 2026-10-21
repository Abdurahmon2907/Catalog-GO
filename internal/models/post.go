package models

type Post struct {
	ID      uint
	Title   string
	Content string
	Image   string
	Author  string
	Likes   int
}