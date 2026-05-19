package model

import (
	"time"
)

type DomainPermission struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID          uint64    `gorm:"index:idx_user_domain,unique;not null" json:"user_id"`
	DomainNodeID    uint64    `gorm:"index:idx_user_domain,unique;not null" json:"domain_node_id"`
	PermissionLevel string    `gorm:"type:varchar(16);default:'read'" json:"permission_level"` // read/write/admin
	AllowedTypes    string    `gorm:"type:text" json:"allowed_types,omitempty"`                // JSON: '["A","AAAA"]', empty=all
	AllowedIPs      string    `gorm:"type:text" json:"allowed_ips,omitempty"`                  // JSON: '["192.168.1.0/24"]', empty=unlimited
	HostPrefix      string    `gorm:"type:varchar(128)" json:"host_prefix,omitempty"`          // e.g. "test-" only allows test-*.domain
	MaxDepth        *int      `json:"max_depth,omitempty"`                                     // max subdomain levels allowed, nil=unlimited
	Status          string    `gorm:"type:varchar(16);default:'active'" json:"status"`         // active/pending_return/returned
	CreatedBy       uint64    `gorm:"not null" json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	User       User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	DomainNode DomainNode `gorm:"foreignKey:DomainNodeID" json:"domain_node,omitempty"`
	Creator    User       `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

func (DomainPermission) TableName() string {
	return "domain_permissions"
}
