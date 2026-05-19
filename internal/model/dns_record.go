package model

import (
	"time"

	"gorm.io/gorm"
)

type DNSRecord struct {
	ID             uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeID         uint64         `gorm:"index;not null" json:"node_id"`
	Host           string         `gorm:"type:varchar(64);not null;default:'@'" json:"host"`
	RecordType     string         `gorm:"type:varchar(10);not null" json:"record_type"`
	Value          string         `gorm:"type:varchar(512);not null" json:"value"`
	TTL            int            `gorm:"default:600" json:"ttl"`
	Priority       *int           `json:"priority,omitempty"`
	Line           string         `gorm:"type:varchar(32);default:'default'" json:"line"`
	Enabled        bool           `gorm:"default:true" json:"enabled"`
	AliyunRecordID string         `gorm:"type:varchar(64)" json:"aliyun_record_id,omitempty"`
	SyncStatus     string         `gorm:"type:varchar(16);default:'pending'" json:"sync_status"`
	LastResolvedAt *time.Time     `json:"last_resolved_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	Node DomainNode `gorm:"foreignKey:NodeID" json:"node,omitempty"`
}

func (DNSRecord) TableName() string {
	return "dns_records"
}
