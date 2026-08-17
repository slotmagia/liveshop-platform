// Package emailsender delivers Platform email test messages through SMTP.
package emailsender

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	emailmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/email/model"
)

type SMTPSender struct {
	dialTimeout time.Duration
}

func New() *SMTPSender {
	return &SMTPSender{dialTimeout: 10 * time.Second}
}

func (s *SMTPSender) Send(ctx context.Context, driver emailmodel.Driver, config map[string]string, input emailmodel.TestSend) (string, error) {
	switch driver {
	case emailmodel.DriverMock:
		return fmt.Sprintf("mock accepted to=%s subject=%s", input.To, input.Subject), nil
	case emailmodel.DriverSMTP:
		return s.sendSMTP(ctx, config, input)
	default:
		return "", emailmodel.ErrInvalid
	}
}

func (s *SMTPSender) sendSMTP(ctx context.Context, config map[string]string, input emailmodel.TestSend) (string, error) {
	host := strings.TrimSpace(config["host"])
	username := strings.TrimSpace(config["username"])
	password := config["password"]
	if host == "" || username == "" || password == "" {
		return "", fmt.Errorf("smtp host/username/password required")
	}
	port := strings.TrimSpace(config["port"])
	if port == "" {
		port = "465"
	}
	fromAddress := strings.TrimSpace(config["from_address"])
	if fromAddress == "" {
		fromAddress = username
	}
	fromName := strings.TrimSpace(config["from_name"])
	addr := net.JoinHostPort(host, port)
	dialer := &net.Dialer{Timeout: s.dialTimeout}
	var client *smtp.Client
	switch strings.TrimSpace(config["encryption"]) {
	case "none":
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return "", fmt.Errorf("connect %s: %w", addr, err)
		}
		client, err = smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return "", fmt.Errorf("connect %s: %w", addr, err)
		}
	case "starttls":
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return "", fmt.Errorf("connect %s: %w", addr, err)
		}
		client, err = smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return "", fmt.Errorf("connect %s: %w", addr, err)
		}
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err = client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
				client.Close()
				return "", fmt.Errorf("starttls %s: %w", addr, err)
			}
		}
	default:
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return "", fmt.Errorf("tls dial %s: %w", addr, err)
		}
		client, err = smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return "", fmt.Errorf("connect %s: %w", addr, err)
		}
	}
	defer client.Close()
	if err := client.Auth(smtp.PlainAuth("", username, password, host)); err != nil {
		return "", fmt.Errorf("auth as %s: %w", username, err)
	}
	if err := client.Mail(fromAddress); err != nil {
		return "", fmt.Errorf("mail from %s: %w", fromAddress, err)
	}
	if err := client.Rcpt(input.To); err != nil {
		return "", fmt.Errorf("rcpt to %s: %w", input.To, err)
	}
	writer, err := client.Data()
	if err != nil {
		return "", fmt.Errorf("data: %w", err)
	}
	from := fromAddress
	if fromName != "" {
		from = mime.QEncoding.Encode("UTF-8", fromName) + " <" + fromAddress + ">"
	}
	message := "From: " + from + "\r\n" +
		"To: " + input.To + "\r\n" +
		"Subject: " + mime.QEncoding.Encode("UTF-8", input.Subject) + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n" +
		"<p>这是一封来自总后台「邮件管理」的测试邮件，收到即说明当前配置可正常发信。</p>"
	if _, err = writer.Write([]byte(message)); err != nil {
		return "", fmt.Errorf("write message: %w", err)
	}
	if err = writer.Close(); err != nil {
		return "", fmt.Errorf("close writer: %w", err)
	}
	if err = client.Quit(); err != nil {
		return "", err
	}
	return "发送成功，请到收件箱确认", nil
}
