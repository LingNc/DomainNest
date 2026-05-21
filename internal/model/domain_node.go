package model

import (
	"time"

	"gorm.io/gorm"
)

type DomainNode struct {
	ID         uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Host       string         `gorm:"type:varchar(64);not null" json:"host"`
	FullDomain string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"full_domain"`
	ParentID   *uint64        `gorm:"index" json:"parent_id"`
	OwnerID    uint64         `gorm:"index;not null" json:"owner_id"`
	ProviderID       *uint64        `gorm:"index" json:"provider_id,omitempty"`
	MaterializedFrom *uint64        `gorm:"index" json:"materialized_from,omitempty"`
	IsMaterialized   bool           `gorm:"default:false" json:"is_materialized"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`

	Parent   *DomainNode   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []DomainNode  `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Owner    User          `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Provider *DNSProvider  `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
	Records  []DNSRecord   `gorm:"foreignKey:NodeID" json:"records,omitempty"`
}

func (DomainNode) TableName() string {
	return "domain_nodes"
}
