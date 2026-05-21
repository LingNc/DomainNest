package model

import "time"

type NodeConversionLog struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	DomainNodeID uint64    `gorm:"index;not null" json:"domain_node_id"`
	Action       string    `gorm:"type:varchar(16);not null" json:"action"` // "materialize" | "dematerialize"
	TriggeredBy  uint64    `gorm:"not null" json:"triggered_by"`
	RecordIDs    string    `gorm:"type:text" json:"record_ids,omitempty"`
	Detail       string    `gorm:"type:text" json:"detail,omitempty"`
	CreatedAt    time.Time `json:"created_at"`

	DomainNode DomainNode `gorm:"foreignKey:DomainNodeID" json:"domain_node,omitempty"`
	Trigger    User       `gorm:"foreignKey:TriggeredBy" json:"trigger,omitempty"`
}

func (NodeConversionLog) TableName() string {
	return "node_conversion_logs"
}
