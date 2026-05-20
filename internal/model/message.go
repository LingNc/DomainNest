package model

import (
	"time"
)

type Message struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	SenderID   uint64     `gorm:"index;not null" json:"sender_id"`
	ReceiverID uint64     `gorm:"index;not null" json:"receiver_id"`
	Type       string     `gorm:"type:varchar(20);default:'user';index" json:"type"`
	Title      string     `gorm:"type:varchar(255)" json:"title"`
	Content    string     `gorm:"type:text;not null" json:"content"`
	ReadAt       *time.Time `gorm:"index" json:"read_at"`
	ActionType   string     `gorm:"type:varchar(30)" json:"action_type"`
	ActionStatus string     `gorm:"type:varchar(20);default:''" json:"action_status"`
	ActionData   string     `gorm:"type:text" json:"action_data"`
	CreatedAt    time.Time  `json:"created_at"`

	Sender   User `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Receiver User `gorm:"foreignKey:ReceiverID" json:"receiver,omitempty"`
}

func (Message) TableName() string {
	return "messages"
}
