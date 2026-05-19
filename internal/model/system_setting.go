package model

import "time"

type SystemSetting struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Key       string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SystemSetting) TableName() string {
	return "system_settings"
}
