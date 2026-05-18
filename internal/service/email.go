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
	cfg *config.SMTPConfig
}

func NewEmailService(cfg *config.SMTPConfig) *EmailService {
	return &EmailService{cfg: cfg}
}

func (s *EmailService) SendPasswordReset(to, resetLink string) {
	if s.cfg.Host == "" || s.cfg.Username == "" {
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
		s.cfg.FromName, s.cfg.From, to, subject, body)

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	if err := smtp.SendMail(addr, auth, s.cfg.From, []string{to}, []byte(msg)); err != nil {
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
