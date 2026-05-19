package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	Password     string         `gorm:"type:varchar(255);not null" json:"-"`
	Email        string         `gorm:"type:varchar(128)" json:"email"`
	Nickname     string         `gorm:"type:varchar(64)" json:"nickname"`
	Phone        string         `gorm:"type:varchar(20)" json:"phone"`
	Avatar       string         `gorm:"type:text" json:"avatar"`
	Role         string         `gorm:"type:enum('admin','user');default:'user'" json:"role"`
	IsSuperAdmin bool           `gorm:"default:false" json:"is_super_admin"`
	Status       int            `gorm:"default:1" json:"status"` // 1=正常 0=禁用
	Token        string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"token,omitempty"`
	InvitedBy    *uint64        `gorm:"index" json:"invited_by"`
	InviteCode   string         `gorm:"type:varchar(32);uniqueIndex" json:"invite_code"`
	InviteLimit  int            `gorm:"default:5" json:"invite_limit"`
	InviteCount  int            `gorm:"default:0" json:"invite_count"`
	LastActiveAt *time.Time     `json:"last_active_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}
