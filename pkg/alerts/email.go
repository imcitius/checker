package alerts

import (
	"bytes"
	"crypto/tls"
	"embed"
	"encoding/json"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"text/template"
	"time"
)

// EmailAlerter implements the Alerter interface for email (SMTP).
type EmailAlerter struct {
	Config EmailConfig
}

func (a *EmailAlerter) Type() string { return "email" }

func (a *EmailAlerter) SendAlert(p AlertPayload) error {
	data := EmailData{
		Subject:      fmt.Sprintf("[ALERT] %s is DOWN", p.CheckName),
		HeaderClass:  "header-down",
		CheckName:    p.CheckName,
		Project:      p.Project,
		CheckType:    p.CheckType,
		ErrorMessage: p.Message,
		Timestamp:    p.Timestamp.Format(time.RFC3339),
	}
	return SendEmailAlert(a.Config, data)
}

func (a *EmailAlerter) SendRecovery(p RecoveryPayload) error {
	data := EmailData{
		Subject:     fmt.Sprintf("[RESOLVED] %s is UP", p.CheckName),
		HeaderClass: "header-up",
		CheckName:   p.CheckName,
		Project:     p.Project,
		CheckType:   p.CheckType,
		Timestamp:   p.Timestamp.Format(time.RFC3339),
	}
	return SendEmailAlert(a.Config, data)
}

func newEmailAlerter(raw json.RawMessage) (Alerter, error) {
	var cfg struct {
		SMTPHost     string   `json:"smtp_host"`
		SMTPPort     int      `json:"smtp_port"`
		SMTPUser     string   `json:"smtp_user"`
		SMTPPassword string   `json:"smtp_password"`
		From         string   `json:"from"`
		To           []string `json:"to"`
		UseTLS       bool     `json:"use_tls"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parsing email config: %w", err)
	}
	if cfg.SMTPHost == "" || cfg.From == "" || len(cfg.To) == 0 {
		return nil, fmt.Errorf("email requires smtp_host, from, and to")
	}
	return &EmailAlerter{Config: EmailConfig{
		SMTPHost:     cfg.SMTPHost,
		SMTPPort:     cfg.SMTPPort,
		SMTPUser:     cfg.SMTPUser,
		SMTPPassword: cfg.SMTPPassword,
		From:         cfg.From,
		To:           cfg.To,
		UseTLS:       cfg.UseTLS,
	}}, nil
}

func init() {
	RegisterAlerter("email", newEmailAlerter)
}

//go:embed templates/email_alert.html templates/email_alert.txt
var emailTemplates embed.FS

// EmailConfig holds SMTP configuration for sending email alerts.
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	From         string
	To           []string
	UseTLS       bool
}

// EmailData holds the template data for rendering email alerts.
type EmailData struct {
	Subject      string
	HeaderClass  string
	CheckName    string
	Project      string
	CheckType    string
	ErrorMessage string
	Timestamp    string
}

// SMTPSender abstracts SMTP sending for testability.
type SMTPSender interface {
	SendMail(addr string, auth smtp.Auth, from string, to []string, msg []byte) error
}

// defaultSMTPSender uses the real net/smtp + TLS logic.
type defaultSMTPSender struct {
	useTLS bool
}

func (s *defaultSMTPSender) SendMail(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address %q: %w", addr, err)
	}

	_, portStr, _ := net.SplitHostPort(addr)

	tlsConfig := &tls.Config{ServerName: host}

	// Port 465: direct TLS (implicit TLS / SMTPS)
	if portStr == "465" {
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS dial failed: %w", err)
		}
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return fmt.Errorf("SMTP client creation failed: %w", err)
		}
		return sendWithClient(client, auth, from, to, msg)
	}

	// Port 587 or others: STARTTLS
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("SMTP dial failed: %w", err)
	}

	if s.useTLS {
		if err := client.StartTLS(tlsConfig); err != nil {
			client.Close()
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
	}

	return sendWithClient(client, auth, from, to, msg)
}

func sendWithClient(client *smtp.Client, auth smtp.Auth, from string, to []string, msg []byte) error {
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			client.Close()
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		client.Close()
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}
	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			client.Close()
			return fmt.Errorf("RCPT TO <%s> failed: %w", addr, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		client.Close()
		return fmt.Errorf("DATA command failed: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		client.Close()
		return fmt.Errorf("writing message failed: %w", err)
	}
	if err := w.Close(); err != nil {
		client.Close()
		return fmt.Errorf("closing data writer failed: %w", err)
	}
	return client.Quit()
}

// smtpSenderInstance is the package-level sender; overridden in tests.
var smtpSenderInstance SMTPSender

// buildEmailMessage constructs the full RFC 2822 message with multipart MIME
// (plain text + HTML).
func buildEmailMessage(cfg EmailConfig, data EmailData) ([]byte, error) {
	txtTmpl, err := template.ParseFS(emailTemplates, "templates/email_alert.txt")
	if err != nil {
		return nil, fmt.Errorf("parsing text template: %w", err)
	}
	htmlTmpl, err := template.ParseFS(emailTemplates, "templates/email_alert.html")
	if err != nil {
		return nil, fmt.Errorf("parsing HTML template: %w", err)
	}

	var txtBuf, htmlBuf bytes.Buffer
	if err := txtTmpl.Execute(&txtBuf, data); err != nil {
		return nil, fmt.Errorf("executing text template: %w", err)
	}
	if err := htmlTmpl.Execute(&htmlBuf, data); err != nil {
		return nil, fmt.Errorf("executing HTML template: %w", err)
	}

	boundary := "----checker-alert-boundary"

	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: %s\r\n", cfg.From))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(cfg.To, ", ")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", mime.QEncoding.Encode("utf-8", data.Subject)))
	msg.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n", boundary))
	msg.WriteString("\r\n")

	// Plain-text part
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	msg.WriteString(txtBuf.String())
	msg.WriteString("\r\n")

	// HTML part
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	msg.WriteString(htmlBuf.String())
	msg.WriteString("\r\n")

	// End boundary
	msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	return msg.Bytes(), nil
}

// SendEmailAlert sends an email alert for a check event.
func SendEmailAlert(cfg EmailConfig, data EmailData) error {
	msgBytes, err := buildEmailMessage(cfg, data)
	if err != nil {
		return fmt.Errorf("building email: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)
	var auth smtp.Auth
	if cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPHost)
	}

	sender := smtpSenderInstance
	if sender == nil {
		sender = &defaultSMTPSender{useTLS: cfg.UseTLS}
	}

	// Extract bare email from "Display Name <email>" format
	from := cfg.From
	if idx := strings.Index(from, "<"); idx != -1 {
		from = strings.Trim(from[idx:], "<>")
	}

	return sender.SendMail(addr, auth, from, cfg.To, msgBytes)
}
