package model

import "time"

type PasswordReset struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"index" json:"user_id"`
	Token     string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `gorm:"default:false" json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

func (PasswordReset) TableName() string {
	return "password_resets"
}
