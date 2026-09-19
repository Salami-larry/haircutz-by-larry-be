package mail

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

const OperationTimeout = 3 * time.Minute

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

type Sender struct {
	cfg Config
}

func NewSender(cfg Config) *Sender {
	return &Sender{cfg: cfg}
}

func (s *Sender) SendHTMLWithPlainAlt(to, subject, plainBody, htmlBody string) error {
	if s.cfg.Host == "" || s.cfg.From == "" {
		return fmt.Errorf("smtp is not configured")
	}
	boundary := "hbl-" + base64.StdEncoding.EncodeToString([]byte(subject+to))[:16]
	var msg strings.Builder
	msg.WriteString("From: " + s.cfg.From + "\r\n")
	msg.WriteString("To: " + to + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: multipart/alternative; boundary=" + boundary + "\r\n")
	msg.WriteString("\r\n")

	writePart := func(contentType, body string) {
		msg.WriteString("--" + boundary + "\r\n")
		msg.WriteString("Content-Type: " + contentType + "\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(body)
		msg.WriteString("\r\n")
	}
	writePart("text/plain; charset=UTF-8", plainBody)
	writePart("text/html; charset=UTF-8", htmlBody)
	msg.WriteString("--" + boundary + "--\r\n")

	return s.sendRaw(to, msg.String())
}

func (s *Sender) sendRaw(to, msg string) error {
	addr := net.JoinHostPort(s.cfg.Host, s.cfg.Port)
	deadline := time.Now().Add(OperationTimeout)

	conn, err := (&net.Dialer{Timeout: OperationTimeout}).Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("smtp deadline: %w", err)
	}

	host := s.cfg.Host
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	if s.cfg.User != "" {
		auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Password, host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp data close: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}

func FormatNGN(kobo int64) string {
	naira := float64(kobo) / 100
	return fmt.Sprintf("₦%.2f", naira)
}

func greetingLine(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Hi,"
	}
	return "Hi " + name + ","
}
