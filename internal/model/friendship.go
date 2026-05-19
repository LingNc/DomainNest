package model

import (
	"time"
)

type Friendship struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"index:idx_user_friend,unique;not null" json:"user_id"`
	FriendID  uint64    `gorm:"index:idx_user_friend,unique;not null" json:"friend_id"`
	CreatedAt time.Time `json:"created_at"`

	User   User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Friend User `gorm:"foreignKey:FriendID" json:"friend,omitempty"`
}

func (Friendship) TableName() string {
	return "friendships"
}
