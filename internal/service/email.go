package service

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"text/template"

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

var resetEmailTmpl = template.Must(template.New("reset").Parse(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background:#f4f6f9;font-family:Arial,Helvetica,sans-serif;">
<table width="100%" cellpadding="0" cellspacing="0" style="background:#f4f6f9;padding:40px 0;">
  <tr><td align="center">
    <table width="520" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;box-shadow:0 2px 12px rgba(0,0,0,0.08);">
      <!-- Header -->
      <tr><td style="background:linear-gradient(135deg,#409eff,#337ecc);padding:28px 32px;border-radius:8px 8px 0 0;">
        <span style="color:#ffffff;font-size:22px;font-weight:700;letter-spacing:1px;">DomainNest</span>
      </td></tr>
      <!-- Body -->
      <tr><td style="padding:32px;">
        <p style="margin:0 0 16px;color:#303133;font-size:16px;">您好，</p>
        <p style="margin:0 0 24px;color:#606266;font-size:14px;line-height:1.6;">您正在进行密码重置操作，验证码如下：</p>
        <table width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding:12px 0 24px;">
          <span style="display:inline-block;background:#f0f7ff;border:1px solid #d9ecff;border-radius:6px;padding:14px 32px;font-size:32px;font-weight:700;color:#409eff;letter-spacing:6px;">{{.Code}}</span>
        </td></tr></table>
        <p style="margin:0 0 8px;color:#909399;font-size:13px;">验证码 <b>{{.ExpiryMinutes}} 分钟</b>内有效，请尽快使用。</p>
        <p style="margin:0;color:#909399;font-size:13px;">如果这不是您的操作，请忽略此邮件，您的账户不会受到影响。</p>
      </td></tr>
      <!-- Footer -->
      <tr><td style="padding:20px 32px;border-top:1px solid #ebeef5;">
        <p style="margin:0;color:#c0c4cc;font-size:12px;text-align:center;">此邮件由系统自动发送，请勿回复</p>
      </td></tr>
    </table>
  </td></tr>
</table>
</body>
</html>`))

func (s *EmailService) SendPasswordReset(to, code string, expiryMinutes int) error {
	cfg := s.getSMTPConfig()
	if cfg == nil || cfg.Host == "" || cfg.Username == "" {
		return fmt.Errorf("SMTP未配置")
	}

	subject := "DomainNest - 密码重置验证码"

	var body bytes.Buffer
	if err := resetEmailTmpl.Execute(&body, struct {
		Code          string
		ExpiryMinutes int
	}{Code: code, ExpiryMinutes: expiryMinutes}); err != nil {
		return fmt.Errorf("模板渲染失败: %w", err)
	}

	msg := fmt.Sprintf("From: %s <%s>\r\n"+
		"To: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"Subject: %s\r\n\r\n%s",
		cfg.FromName, cfg.From, to, subject, body.String())

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
		return fmt.Errorf("SMTP未配置")
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

var verifyEmailTmpl = template.Must(template.New("verify").Parse(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background:#f4f6f9;font-family:Arial,Helvetica,sans-serif;">
<table width="100%" cellpadding="0" cellspacing="0" style="background:#f4f6f9;padding:40px 0;">
  <tr><td align="center">
    <table width="520" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;box-shadow:0 2px 12px rgba(0,0,0,0.08);">
      <tr><td style="background:linear-gradient(135deg,#67c23a,#529b2e);padding:28px 32px;border-radius:8px 8px 0 0;">
        <span style="color:#ffffff;font-size:22px;font-weight:700;letter-spacing:1px;">DomainNest</span>
      </td></tr>
      <tr><td style="padding:32px;">
        <p style="margin:0 0 16px;color:#303133;font-size:16px;">您好，</p>
        <p style="margin:0 0 24px;color:#606266;font-size:14px;line-height:1.6;">您正在进行邮箱验证操作，验证码如下：</p>
        <table width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding:12px 0 24px;">
          <span style="display:inline-block;background:#f0f9eb;border:1px solid #e1f3d8;border-radius:6px;padding:14px 32px;font-size:32px;font-weight:700;color:#67c23a;letter-spacing:6px;">{{.Code}}</span>
        </td></tr></table>
        <p style="margin:0 0 8px;color:#909399;font-size:13px;">验证码 <b>5 分钟</b>内有效，请尽快使用。</p>
        <p style="margin:0;color:#909399;font-size:13px;">如果这不是您的操作，请忽略此邮件。</p>
      </td></tr>
      <tr><td style="padding:20px 32px;border-top:1px solid #ebeef5;">
        <p style="margin:0;color:#c0c4cc;font-size:12px;text-align:center;">此邮件由系统自动发送，请勿回复</p>
      </td></tr>
    </table>
  </td></tr>
</table>
</body>
</html>`))

func (s *EmailService) SendEmailVerification(to, code string) error {
	cfg := s.getSMTPConfig()
	if cfg == nil || cfg.Host == "" || cfg.Username == "" {
		return fmt.Errorf("SMTP未配置")
	}

	subject := "DomainNest - 邮箱验证码"

	var body bytes.Buffer
	if err := verifyEmailTmpl.Execute(&body, struct{ Code string }{Code: code}); err != nil {
		return fmt.Errorf("模板渲染失败: %w", err)
	}

	msg := fmt.Sprintf("From: %s <%s>\r\n"+
		"To: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"Subject: %s\r\n\r\n%s",
		cfg.FromName, cfg.From, to, subject, body.String())

	if err := sendMail(cfg, []string{to}, []byte(msg)); err != nil {
		log.Printf("[Email] Failed to send verification email to %s: %v", to, err)
		return err
	}
	log.Printf("[Email] Verification email sent to %s", to)
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
