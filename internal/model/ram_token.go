package model

import (
	"time"

	"gorm.io/gorm"
)

type RAMToken struct {
	ID             uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         uint64         `gorm:"index;not null" json:"user_id"`
	Name           string         `gorm:"type:varchar(64);not null" json:"name"`
	Token          string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"token"`
	Enabled        bool           `gorm:"default:true" json:"enabled"`
	AllowedDomains string         `gorm:"type:text" json:"allowed_domains,omitempty"` // JSON: [1,2,3] domain_node_ids, empty=all
	AllowedTypes   string         `gorm:"type:text" json:"allowed_types,omitempty"`   // JSON: ["A","AAAA"], empty=all
	AllowedIPs     string         `gorm:"type:text" json:"allowed_ips,omitempty"`     // JSON: ["192.168.1.0/24"], empty=unlimited
	UsageCount     int64          `gorm:"default:0" json:"usage_count"`
	LastUsedAt     *time.Time     `json:"last_used_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (RAMToken) TableName() string {
	return "ram_tokens"
}
