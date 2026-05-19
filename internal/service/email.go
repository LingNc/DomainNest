package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/smtp"

	"domainnest/internal/config"
)

type EmailService struct {
	cfg      *config.SMTPConfig
	settings *SettingsService
}

func NewEmailService(cfg *config.SMTPConfig) *EmailService {
	return &EmailService{cfg: cfg}
}

func NewEmailServiceWithSettings(cfg *config.SMTPConfig, settings *SettingsService) *EmailService {
	return &EmailService{cfg: cfg, settings: settings}
}

func (s *EmailService) getSMTPConfig() *config.SMTPConfig {
	if s.settings != nil {
		if cfg := s.settings.GetSMTPConfig(); cfg != nil {
			return cfg
		}
	}
	return s.cfg
}

func (s *EmailService) SendPasswordReset(to, resetLink string) {
	cfg := s.getSMTPConfig()
	if cfg == nil || cfg.Host == "" || cfg.Username == "" {
		log.Printf("[Email] SMTP not configured, skip sending reset email to %s", to)
		return
	}

	subject := "DomainNest - Password Reset"
	body := fmt.Sprintf("Hello,<br><br>"+
		"You requested a password reset. Click the link below to reset your password:<br><br>"+
		"<a href=\"%s\">%s</a><br><br>"+
		"This link will expire in 30 minutes.<br><br>"+
		"If you did not request this, please ignore this email.", resetLink, resetLink)

	msg := fmt.Sprintf("From: %s <%s>\r\n"+
		"To: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"Subject: %s\r\n\r\n%s",
		cfg.FromName, cfg.From, to, subject, body)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	if err := smtp.SendMail(addr, auth, cfg.From, []string{to}, []byte(msg)); err != nil {
		log.Printf("[Email] Failed to send reset email to %s: %v", to, err)
	} else {
		log.Printf("[Email] Reset email sent to %s", to)
	}
}

func GenerateToken(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
