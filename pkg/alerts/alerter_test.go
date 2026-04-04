package alerts

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAlertPayload_Fields(t *testing.T) {
	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	p := AlertPayload{
		CheckName:  "http-check",
		CheckUUID:  "uuid-123",
		Project:    "myproject",
		CheckGroup: "web",
		CheckType:  "http",
		Message:    "connection refused",
		Severity:   "critical",
		Timestamp:  ts,
	}
	assert.Equal(t, "http-check", p.CheckName)
	assert.Equal(t, "uuid-123", p.CheckUUID)
	assert.Equal(t, "myproject", p.Project)
	assert.Equal(t, "web", p.CheckGroup)
	assert.Equal(t, "http", p.CheckType)
	assert.Equal(t, "connection refused", p.Message)
	assert.Equal(t, "critical", p.Severity)
	assert.Equal(t, ts, p.Timestamp)
}

func TestRecoveryPayload_Fields(t *testing.T) {
	ts := time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC)
	p := RecoveryPayload{
		CheckName:  "tcp-check",
		CheckUUID:  "uuid-456",
		Project:    "myproject",
		CheckGroup: "infra",
		CheckType:  "tcp",
		Timestamp:  ts,
	}
	assert.Equal(t, "tcp-check", p.CheckName)
	assert.Equal(t, "uuid-456", p.CheckUUID)
	assert.Equal(t, "myproject", p.Project)
	assert.Equal(t, "infra", p.CheckGroup)
	assert.Equal(t, "tcp", p.CheckType)
	assert.Equal(t, ts, p.Timestamp)
}

func TestAlerterInterface_StubCompliance(t *testing.T) {
	// Verify that a minimal stub satisfies the Alerter interface at compile time.
	var a Alerter = &stubAlerter{channelType: "test"}
	assert.Equal(t, "test", a.Type())
	assert.NoError(t, a.SendAlert(AlertPayload{}))
	assert.NoError(t, a.SendRecovery(RecoveryPayload{}))
}

func TestAlertPayload_ZeroValue(t *testing.T) {
	var p AlertPayload
	assert.Empty(t, p.CheckName)
	assert.Empty(t, p.Message)
	assert.Empty(t, p.Severity)
	assert.True(t, p.Timestamp.IsZero())
}

func TestRecoveryPayload_ZeroValue(t *testing.T) {
	var p RecoveryPayload
	assert.Empty(t, p.CheckName)
	assert.Empty(t, p.CheckUUID)
	assert.True(t, p.Timestamp.IsZero())
}
