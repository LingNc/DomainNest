package model

import (
	"time"
)

type Message struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	SenderID   uint64     `gorm:"index;not null" json:"sender_id"`
	ReceiverID uint64     `gorm:"index:idx_receiver_category;not null" json:"receiver_id"`
	Type       string     `gorm:"type:varchar(20);default:'user';index" json:"type"`
	Title      string     `gorm:"type:varchar(255)" json:"title"`
	Content    string     `gorm:"type:text;not null" json:"content"`
	ReadAt       *time.Time `gorm:"index" json:"read_at"`
	ActionType   string     `gorm:"type:varchar(30)" json:"action_type"`
	ActionStatus string     `gorm:"type:varchar(20);default:''" json:"action_status"`
	ActionData   string     `gorm:"type:text" json:"action_data"`
	Category     string     `gorm:"type:varchar(40);default:'';index:idx_receiver_category" json:"category"`
	Priority     int        `gorm:"default:0" json:"priority"`
	TargetType   string     `gorm:"type:varchar(30);default:''" json:"target_type"`
	TargetID     uint64     `gorm:"default:0" json:"target_id"`
	ExpiresAt    *time.Time `gorm:"index" json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`

	Sender   User `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Receiver User `gorm:"foreignKey:ReceiverID" json:"receiver,omitempty"`
}

func (Message) TableName() string {
	return "messages"
}
