package model

import "time"

type Post struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Content   string    `gorm:"not null;size:2000" json:"content"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"author"`
	Comments  []Comment `gorm:"foreignKey:PostID" json:"comments,omitempty"`
	LikeCount int       `gorm:"default:0" json:"like_count"`
	ViewCount int       `gorm:"default:0" json:"view_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PostResponse struct {
	ID           uint         `json:"id"`
	Content      string       `json:"content"`
	Author       UserResponse `json:"author"`
	LikeCount    int          `json:"like_count"`
	CommentCount int64        `json:"comment_count"`
	ViewCount    int          `json:"view_count"`
	CreatedAt    time.Time    `json:"created_at"`
}

type PostDetailResponse struct {
	PostResponse
	Comments []CommentResponse `json:"comments"`
}
