package model

import "time"

type AgentMessage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SessionID string    `gorm:"not null;size:128;index" json:"session_id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Role      string    `gorm:"not null;size:20" json:"role"`
	Content   string    `gorm:"not null;size:4000" json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type AgentDraft struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	DraftID   string    `gorm:"unique;not null;size:64" json:"draft_id"`
	SessionID string    `gorm:"not null;size:128;index" json:"session_id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Action    string    `gorm:"not null;size:50" json:"action"`
	Content   string    `gorm:"not null;size:2000" json:"content"`
	Status    string    `gorm:"not null;default:pending;size:20" json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PendingActionResponse struct {
	DraftID   string    `json:"draft_id"`
	Action    string    `json:"action"`
	Content   string    `json:"content"`
	ExpiresAt time.Time `json:"expires_at"`
}
