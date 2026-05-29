package model

import "time"

type AcmeDNSAccount struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint64    `gorm:"index;not null" json:"user_id"`
	Username   string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	Password   string    `gorm:"type:varchar(128);not null" json:"-"`
	Subdomain  string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"subdomain"`
	FullDomain string    `gorm:"type:varchar(253);not null" json:"full_domain"`
	NodeID     uint64    `gorm:"index;not null" json:"node_id"`
	CreatedAt  time.Time `json:"created_at"`
}

func (AcmeDNSAccount) TableName() string {
	return "acmedns_accounts"
}