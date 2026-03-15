package alerts

import (
	"net/smtp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSMTPSender captures the arguments passed to SendMail for assertions.
type mockSMTPSender struct {
	from    string
	to      []string
	msg     []byte
	err     error
	called  bool
}

func (m *mockSMTPSender) SendMail(_ string, _ smtp.Auth, from string, to []string, msg []byte) error {
	m.called = true
	m.from = from
	m.to = to
	m.msg = msg
	return m.err
}

func baseCfg() EmailConfig {
	return EmailConfig{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUser:     "alerts@example.com",
		SMTPPassword: "secret",
		From:         "Checker Alerts <alerts@example.com>",
		To:           []string{"ops@example.com"},
		UseTLS:       true,
	}
}

func TestSendEmailAlert_DownSubject(t *testing.T) {
	mock := &mockSMTPSender{}
	smtpSenderInstance = mock
	defer func() { smtpSenderInstance = nil }()

	data := EmailData{
		Subject:      "[ALERT] my-api is DOWN",
		HeaderClass:  "header-down",
		CheckName:    "my-api",
		Project:      "backend",
		CheckType:    "http",
		ErrorMessage: "connection refused",
		Timestamp:    "2026-03-15T10:00:00Z",
	}

	err := SendEmailAlert(baseCfg(), data)
	require.NoError(t, err)
	assert.True(t, mock.called)

	msgStr := string(mock.msg)

	// Verify subject contains ALERT / DOWN
	assert.Contains(t, msgStr, "Subject:")
	assert.Contains(t, msgStr, "ALERT")
	assert.Contains(t, msgStr, "DOWN")

	// Verify body content
	assert.Contains(t, msgStr, "my-api")
	assert.Contains(t, msgStr, "backend")
	assert.Contains(t, msgStr, "connection refused")
	assert.Contains(t, msgStr, "2026-03-15T10:00:00Z")
}

func TestSendEmailAlert_ResolvedSubject(t *testing.T) {
	mock := &mockSMTPSender{}
	smtpSenderInstance = mock
	defer func() { smtpSenderInstance = nil }()

	data := EmailData{
		Subject:     "[RESOLVED] my-api is UP",
		HeaderClass: "header-up",
		CheckName:   "my-api",
		Project:     "backend",
		CheckType:   "http",
		Timestamp:   "2026-03-15T10:05:00Z",
	}

	err := SendEmailAlert(baseCfg(), data)
	require.NoError(t, err)
	assert.True(t, mock.called)

	msgStr := string(mock.msg)

	// Verify subject contains RESOLVED / UP
	assert.Contains(t, msgStr, "Subject:")
	assert.Contains(t, msgStr, "RESOLVED")
	assert.Contains(t, msgStr, "UP")

	// Verify no error message block in text part
	// The text template uses {{- if .ErrorMessage}} so it should be absent
	assert.NotContains(t, msgStr, "Error:")
}

func TestSendEmailAlert_FromAddressExtraction(t *testing.T) {
	mock := &mockSMTPSender{}
	smtpSenderInstance = mock
	defer func() { smtpSenderInstance = nil }()

	data := EmailData{
		Subject:   "[ALERT] test is DOWN",
		CheckName: "test",
		Project:   "p",
		CheckType: "tcp",
		Timestamp: "2026-03-15T10:00:00Z",
	}

	err := SendEmailAlert(baseCfg(), data)
	require.NoError(t, err)

	// From should be the bare email extracted from "Display <email>"
	assert.Equal(t, "alerts@example.com", mock.from)
}

func TestSendEmailAlert_MultipleRecipients(t *testing.T) {
	mock := &mockSMTPSender{}
	smtpSenderInstance = mock
	defer func() { smtpSenderInstance = nil }()

	cfg := baseCfg()
	cfg.To = []string{"ops@example.com", "dev@example.com"}

	data := EmailData{
		Subject:   "[ALERT] db is DOWN",
		CheckName: "db",
		Project:   "infra",
		CheckType: "tcp",
		Timestamp: "2026-03-15T10:00:00Z",
	}

	err := SendEmailAlert(cfg, data)
	require.NoError(t, err)
	assert.Equal(t, []string{"ops@example.com", "dev@example.com"}, mock.to)

	// Both recipients should appear in the To header
	msgStr := string(mock.msg)
	assert.Contains(t, msgStr, "ops@example.com")
	assert.Contains(t, msgStr, "dev@example.com")
}

func TestSendEmailAlert_SMTPError(t *testing.T) {
	mock := &mockSMTPSender{err: assert.AnError}
	smtpSenderInstance = mock
	defer func() { smtpSenderInstance = nil }()

	data := EmailData{
		Subject:   "[ALERT] test is DOWN",
		CheckName: "test",
		Project:   "p",
		CheckType: "http",
		Timestamp: "now",
	}

	err := SendEmailAlert(baseCfg(), data)
	assert.Error(t, err)
}

func TestBuildEmailMessage_Multipart(t *testing.T) {
	cfg := baseCfg()
	data := EmailData{
		Subject:      "[ALERT] web is DOWN",
		HeaderClass:  "header-down",
		CheckName:    "web",
		Project:      "frontend",
		CheckType:    "http",
		ErrorMessage: "timeout",
		Timestamp:    "2026-03-15T10:00:00Z",
	}

	msg, err := buildEmailMessage(cfg, data)
	require.NoError(t, err)

	msgStr := string(msg)

	// Should be multipart/alternative
	assert.Contains(t, msgStr, "multipart/alternative")

	// Should contain both text/plain and text/html parts
	assert.Contains(t, msgStr, "Content-Type: text/plain")
	assert.Contains(t, msgStr, "Content-Type: text/html")

	// HTML part should have the header class
	assert.Contains(t, msgStr, "header-down")

	// Count boundary markers — should have opening + closing
	boundary := "----checker-alert-boundary"
	assert.True(t, strings.Count(msgStr, boundary) >= 3, "expected at least 3 boundary markers")
}
