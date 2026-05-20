package model

import (
	"time"
)

type OperationLog struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint64    `gorm:"index;not null" json:"user_id"`
	TargetUserID *uint64   `gorm:"index" json:"target_user_id,omitempty"`
	Action       string    `gorm:"type:varchar(32);not null" json:"action"`
	TargetType   string    `gorm:"type:varchar(32)" json:"target_type"`
	TargetID     *uint64   `json:"target_id,omitempty"`
	Detail       string    `gorm:"type:text" json:"detail"`
	IPAddress    string    `gorm:"type:varchar(64)" json:"ip_address"`
	CreatedAt    time.Time `json:"created_at"`

	User       User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	TargetUser *User `gorm:"foreignKey:TargetUserID" json:"target_user,omitempty"`
}

func (OperationLog) TableName() string {
	return "operation_logs"
}
