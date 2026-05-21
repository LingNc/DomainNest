package model

import "time"

type FilterPreset struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `gorm:"index;not null" json:"user_id"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	Filters   string    `gorm:"type:text;not null" json:"filters"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
