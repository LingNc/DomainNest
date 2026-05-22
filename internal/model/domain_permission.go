package model

import (
	"time"
)

// HostRuleType defines how a host value is matched.
type HostRuleType string

const (
	HostRuleExact    HostRuleType = "exact"    // "www" matches only "www"
	HostRulePrefix   HostRuleType = "prefix"   // "test-" matches "test-*"
	HostRuleSuffix   HostRuleType = "suffix"   // "-prod" matches "*-prod"
	HostRuleContains HostRuleType = "contains" // "api" matches "*api*"
	HostRuleRegex    HostRuleType = "regex"    // full regex pattern
)

// HostRule is a single host restriction rule.
type HostRule struct {
	Type  HostRuleType `json:"type"`  // exact|prefix|suffix|contains|regex
	Value string       `json:"value"` // the pattern string
}

type DomainPermission struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID          uint64    `gorm:"index:idx_user_domain,unique;not null" json:"user_id"`
	DomainNodeID    uint64    `gorm:"index:idx_user_domain,unique;not null" json:"domain_node_id"`
	PermissionLevel string    `gorm:"type:varchar(16);default:'read'" json:"permission_level"` // read/write/admin
	AllowedTypes    string    `gorm:"type:text" json:"allowed_types,omitempty"`                // JSON: '["A","AAAA"]', empty=all
	AllowedIPs      string    `gorm:"type:text" json:"allowed_ips,omitempty"`                  // JSON: '["192.168.1.0/24"]', empty=unlimited
	HostPrefix      string    `gorm:"type:varchar(128)" json:"host_prefix,omitempty"`          // e.g. "test-" only allows test-*.domain
	HostRules       string    `gorm:"type:text" json:"host_rules,omitempty"`                   // JSON array of HostRule, e.g. [{"type":"prefix","value":"test-"}]
	MaxDepth        *int      `json:"max_depth,omitempty"`                                     // max subdomain levels allowed, nil=unlimited
	SourceFilter   *string   `gorm:"type:varchar(20)" json:"source_filter,omitempty"`           // "provider"|"platform"|nil(none)
	Status         string    `gorm:"type:varchar(20);default:'active';index" json:"status"`    // active/frozen/pending_return/returned
	CreatedBy      uint64    `gorm:"not null" json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	User       User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	DomainNode DomainNode `gorm:"foreignKey:DomainNodeID" json:"domain_node,omitempty"`
	Creator    User       `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

func (DomainPermission) TableName() string {
	return "domain_permissions"
}
