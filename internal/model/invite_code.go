package model

import "time"

type InviteCode struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Code      string     `gorm:"type:varchar(20);uniqueIndex;not null" json:"code"`
	CreatorID uint64     `gorm:"index;not null" json:"creator_id"`
	UsedBy    *uint64    `gorm:"index" json:"used_by,omitempty"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}
