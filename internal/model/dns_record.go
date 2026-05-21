package model

import (
	"time"

	"gorm.io/gorm"
)

type DNSRecord struct {
	ID               uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeID           uint64         `gorm:"index;not null" json:"node_id"`
	OwnNodeID        *uint64        `gorm:"index" json:"own_node_id,omitempty"`
	Host             string         `gorm:"type:varchar(64);not null;default:'@'" json:"host"`
	RecordType       string         `gorm:"type:varchar(10);not null" json:"record_type"`
	Value            string         `gorm:"type:varchar(512);not null" json:"value"`
	TTL              int            `gorm:"default:600" json:"ttl"`
	Priority         *int           `json:"priority,omitempty"`
	Line             string         `gorm:"type:varchar(32);default:'default'" json:"line"`
	Enabled          bool           `gorm:"default:true" json:"enabled"`
	ProviderRecordID string         `gorm:"type:varchar(128);column:provider_record_id" json:"provider_record_id,omitempty"`
	SyncStatus       string         `gorm:"type:varchar(16);default:'pending'" json:"sync_status"`
	LastSyncError    string         `gorm:"type:text" json:"last_sync_error,omitempty"`
	SyncAttempts     int            `gorm:"default:0" json:"sync_attempts"`
	NextSyncAt       *time.Time     `gorm:"index" json:"next_sync_at,omitempty"`
	PendingGroup     string         `gorm:"type:varchar(64);index" json:"pending_group,omitempty"`
	CreatedBy        uint64         `gorm:"index" json:"created_by,omitempty"`
	LastResolvedAt   *time.Time     `json:"last_resolved_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`

	Node DomainNode `gorm:"foreignKey:NodeID" json:"node,omitempty"`
}

func (DNSRecord) TableName() string {
	return "dns_records"
}
