package service

import (
	"time"

	"domainnest/internal/model"

	"gorm.io/gorm"
)

type EmailVerifyService struct {
	db    *gorm.DB
	email *EmailService
}

func NewEmailVerifyService(db *gorm.DB, email *EmailService) *EmailVerifyService {
	return &EmailVerifyService{db: db, email: email}
}

// SendCode generates and sends a verification code, storing it in the database.
func (s *EmailVerifyService) SendCode(email, purpose string) error {
	code, err := GenerateVerifyCode()
	if err != nil {
		return err
	}

	// Invalidate previous codes for this email+purpose
	s.db.Model(&model.EmailVerification{}).
		Where("email = ? AND purpose = ? AND used = false", email, purpose).
		Update("used", true)

	ev := model.EmailVerification{
		Email:     email,
		Code:      code,
		Purpose:   purpose,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	if err := s.db.Create(&ev).Error; err != nil {
		return err
	}

	return s.email.SendEmailVerification(email, code)
}

// VerifyCode checks the code and marks it as used. Returns true if valid.
func (s *EmailVerifyService) VerifyCode(email, code, purpose string) bool {
	var ev model.EmailVerification
	err := s.db.Where("email = ? AND code = ? AND purpose = ? AND used = false AND expires_at > ?",
		email, code, purpose, time.Now()).First(&ev).Error
	if err != nil {
		return false
	}
	s.db.Model(&ev).Update("used", true)
	return true
}
