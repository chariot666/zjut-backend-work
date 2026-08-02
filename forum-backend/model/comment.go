package model

import "time"

type Comment struct {
	ID uint `gorm:"primaryKey" json:"id"`

	Content string `json:"content"`

	UserID uint `json:"user_id"`

	PostID uint `json:"post_id"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`
}
