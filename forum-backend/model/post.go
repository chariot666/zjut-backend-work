package model

import "time"


type Post struct {

	ID uint `gorm:"primaryKey"`


	Title string


	Content string


	UserID uint


	ViewCount int


	LikeCount int


	CreatedAt time.Time


	UpdatedAt time.Time
}