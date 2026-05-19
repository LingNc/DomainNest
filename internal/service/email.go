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

func (s *EmailService) SendPasswordReset(to, code string) {
	cfg := s.getSMTPConfig()
	if cfg == nil || cfg.Host == "" || cfg.Username == "" {
		log.Printf("[Email] SMTP not configured, skip sending reset email to %s", to)
		return
	}

	subject := "DomainNest - 密码重置验证码"
	body := fmt.Sprintf("您好，<br><br>"+
		"您正在进行密码重置操作，验证码如下：<br><br>"+
		"<h2 style=\"color:#409eff;letter-spacing:4px\">%s</h2><br>"+
		"验证码 30 分钟内有效。<br><br>"+
		"如果这不是您的操作，请忽略此邮件。", code)

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

func GenerateVerifyCode() (string, error) {
	bytes := make([]byte, 3)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	n := int(bytes[0])<<16 | int(bytes[1])<<8 | int(bytes[2])
	return fmt.Sprintf("%06d", n%1000000), nil
}

func GenerateToken(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
