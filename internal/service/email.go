package service

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"

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

// hostname extracts just the hostname from a host or host:port string.
func hostname(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

// sendMail sends an email using the appropriate TLS method based on config.
func sendMail(cfg *config.SMTPConfig, to []string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	host := hostname(cfg.Host)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, host)

	tlsType := strings.ToLower(cfg.TLSType)
	if tlsType == "" {
		tlsType = "starttls"
	}

	switch tlsType {
	case "implicit":
		// Implicit TLS (port 465): dial with TLS from the start
		tlsConfig := &tls.Config{ServerName: host}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS dial failed: %w", err)
		}
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return fmt.Errorf("SMTP client creation failed: %w", err)
		}
		defer client.Close()

		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
		if err = client.Mail(cfg.From); err != nil {
			return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
		}
		for _, addr := range to {
			if err = client.Rcpt(addr); err != nil {
				return fmt.Errorf("SMTP RCPT TO failed: %w", err)
			}
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("SMTP DATA failed: %w", err)
		}
		if _, err = w.Write(msg); err != nil {
			return fmt.Errorf("SMTP write failed: %w", err)
		}
		if err = w.Close(); err != nil {
			return fmt.Errorf("SMTP data close failed: %w", err)
		}
		return client.Quit()

	case "none":
		// No TLS, no auth
		return smtp.SendMail(addr, nil, cfg.From, to, msg)

	default:
		// STARTTLS (default, port 587)
		return smtp.SendMail(addr, auth, cfg.From, to, msg)
	}
}

func (s *EmailService) SendPasswordReset(to, code string) error {
	cfg := s.getSMTPConfig()
	if cfg == nil || cfg.Host == "" || cfg.Username == "" {
		return fmt.Errorf("SMTP not configured")
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

	if err := sendMail(cfg, []string{to}, []byte(msg)); err != nil {
		log.Printf("[Email] Failed to send reset email to %s: %v", to, err)
		return err
	}
	log.Printf("[Email] Reset email sent to %s", to)
	return nil
}

func (s *EmailService) SendTestEmail(to string) error {
	cfg := s.getSMTPConfig()
	if cfg == nil || cfg.Host == "" || cfg.Username == "" {
		return fmt.Errorf("SMTP not configured")
	}

	subject := "DomainNest SMTP Test"
	body := "This is a test email from DomainNest."

	msg := fmt.Sprintf("From: %s <%s>\r\n"+
		"To: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"Subject: %s\r\n\r\n%s",
		cfg.FromName, cfg.From, to, subject, body)

	if err := sendMail(cfg, []string{to}, []byte(msg)); err != nil {
		log.Printf("[Email] Failed to send test email to %s: %v", to, err)
		return err
	}
	log.Printf("[Email] Test email sent to %s", to)
	return nil
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
