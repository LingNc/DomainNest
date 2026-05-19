package model

import (
	"time"
)

type InviteLog struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	InviterID uint64    `gorm:"index;not null" json:"inviter_id"`
	InviteeID uint64    `gorm:"index;not null" json:"invitee_id"`
	Action    string    `gorm:"type:varchar(32);not null" json:"action"` // register/grant/admin_grant
	Amount    int       `gorm:"default:1" json:"amount"`
	CreatedAt time.Time `json:"created_at"`

	Inviter User `gorm:"foreignKey:InviterID" json:"inviter,omitempty"`
	Invitee User `gorm:"foreignKey:InviteeID" json:"invitee,omitempty"`
}

func (InviteLog) TableName() string {
	return "invite_logs"
}
