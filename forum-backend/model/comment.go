package model

import "time"

type Comment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Content   string    `gorm:"not null;size:1000" json:"content"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"author"`
	PostID    uint      `gorm:"not null;index" json:"post_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CommentResponse struct {
	ID        uint         `json:"id"`
	PostID    uint         `json:"post_id"`
	Content   string       `json:"content"`
	Author    UserResponse `json:"author"`
	CreatedAt time.Time    `json:"created_at"`
}

func (c Comment) ToResponse() CommentResponse {
	return CommentResponse{
		ID:        c.ID,
		PostID:    c.PostID,
		Content:   c.Content,
		Author:    c.User.ToResponse(),
		CreatedAt: c.CreatedAt,
	}
}
