package model

import "time"

type DNSProvider struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID          uint64    `gorm:"index;not null" json:"user_id"`
	ProviderType    string    `gorm:"type:varchar(32);not null" json:"provider_type"` // "aliyun"
	Name            string    `gorm:"type:varchar(64);not null" json:"name"`
	AccessKeyID     string    `gorm:"type:varchar(128);not null" json:"access_key_id"`
	AccessKeySecret string    `gorm:"type:varchar(255);not null" json:"-"`
	Endpoint        string    `gorm:"type:varchar(128)" json:"endpoint,omitempty"`
	Status          string    `gorm:"type:varchar(16);default:'active'" json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (DNSProvider) TableName() string {
	return "dns_providers"
}
