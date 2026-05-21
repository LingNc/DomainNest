package model

import "time"

type SyncLog struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RecordID   uint64    `gorm:"index;not null" json:"record_id"`
	Action     string    `gorm:"type:varchar(32);not null" json:"action"`  // "create"|"update"|"delete"
	Status     string    `gorm:"type:varchar(16);not null" json:"status"`  // "success"|"failed"
	Error      string    `gorm:"type:text" json:"error,omitempty"`
	ProviderID string    `gorm:"type:varchar(128)" json:"provider_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`

	Record DNSRecord `gorm:"foreignKey:RecordID" json:"record,omitempty"`
}

func (SyncLog) TableName() string {
	return "sync_logs"
}
