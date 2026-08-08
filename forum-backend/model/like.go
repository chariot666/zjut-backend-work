package model

import "time"

type Like struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_user_post_like" json:"user_id"`
	PostID    uint      `gorm:"not null;uniqueIndex:idx_user_post_like" json:"post_id"`
	CreatedAt time.Time `json:"created_at"`
}
