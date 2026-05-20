package model

import "time"

type EmailVerification struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Email     string    `gorm:"type:varchar(128);index;not null" json:"email"`
	Code      string    `gorm:"type:varchar(10);not null" json:"-"`
	Purpose   string    `gorm:"type:varchar(30);not null" json:"purpose"` // register, change_email
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	Used      bool      `gorm:"default:false" json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

func (EmailVerification) TableName() string {
	return "email_verifications"
}
