package model

import (
	"time"
)

type FriendRequest struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	SenderID   uint64    `gorm:"index:idx_sender_receiver,unique;not null" json:"sender_id"`
	ReceiverID uint64    `gorm:"index:idx_sender_receiver,unique;not null" json:"receiver_id"`
	Status     string    `gorm:"type:enum('pending','accepted','rejected');default:'pending'" json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	Sender   User `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Receiver User `gorm:"foreignKey:ReceiverID" json:"receiver,omitempty"`
}

func (FriendRequest) TableName() string {
	return "friend_requests"
}
