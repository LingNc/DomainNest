package model

import "time"

type DomainTransferLog struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeID     uint64    `gorm:"index;not null" json:"node_id"`
	FromUserID uint64    `gorm:"index;not null" json:"from_user_id"`
	ToUserID   uint64    `gorm:"index;not null" json:"to_user_id"`
	CreatedAt  time.Time `json:"created_at"`

	Node     DomainNode `gorm:"foreignKey:NodeID" json:"node,omitempty"`
	FromUser User       `gorm:"foreignKey:FromUserID" json:"from_user,omitempty"`
	ToUser   User       `gorm:"foreignKey:ToUserID" json:"to_user,omitempty"`
}

func (DomainTransferLog) TableName() string {
	return "domain_transfer_logs"
}
