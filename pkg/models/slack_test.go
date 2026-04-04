package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSlackAlertThread_ZeroValue(t *testing.T) {
	var s SlackAlertThread
	assert.Equal(t, 0, s.ID)
	assert.Empty(t, s.CheckUUID)
	assert.Empty(t, s.ChannelID)
	assert.Empty(t, s.ThreadTs)
	assert.Empty(t, s.ParentTs)
	assert.False(t, s.IsResolved)
	assert.True(t, s.CreatedAt.IsZero())
	assert.Nil(t, s.ResolvedAt)
}

func TestSlackAlertThread_WithFields(t *testing.T) {
	now := time.Now()
	resolved := now.Add(5 * time.Minute)
	s := SlackAlertThread{
		ID:         1,
		CheckUUID:  "uuid-slack-1",
		ChannelID:  "C0123ABC",
		ThreadTs:   "1234567890.123456",
		ParentTs:   "1234567890.000000",
		IsResolved: true,
		CreatedAt:  now,
		ResolvedAt: &resolved,
	}

	assert.Equal(t, 1, s.ID)
	assert.Equal(t, "uuid-slack-1", s.CheckUUID)
	assert.Equal(t, "C0123ABC", s.ChannelID)
	assert.Equal(t, "1234567890.123456", s.ThreadTs)
	assert.Equal(t, "1234567890.000000", s.ParentTs)
	assert.True(t, s.IsResolved)
	assert.NotNil(t, s.ResolvedAt)
}

func TestSlackAlertThread_UnresolvedThread(t *testing.T) {
	s := SlackAlertThread{
		ID:         2,
		CheckUUID:  "uuid-slack-2",
		ChannelID:  "C0456DEF",
		ThreadTs:   "9876543210.654321",
		IsResolved: false,
		ResolvedAt: nil,
	}
	assert.False(t, s.IsResolved)
	assert.Nil(t, s.ResolvedAt)
}

func TestAlertSilence_ZeroValue(t *testing.T) {
	var s AlertSilence
	assert.Equal(t, 0, s.ID)
	assert.Empty(t, s.Scope)
	assert.Empty(t, s.Target)
	assert.Empty(t, s.Channel)
	assert.Empty(t, s.SilencedBy)
	assert.True(t, s.SilencedAt.IsZero())
	assert.Nil(t, s.ExpiresAt)
	assert.Empty(t, s.Reason)
	assert.False(t, s.Active)
}

func TestAlertSilence_WithFields(t *testing.T) {
	now := time.Now()
	expires := now.Add(1 * time.Hour)
	s := AlertSilence{
		ID:         1,
		Scope:      "check",
		Target:     "uuid-123",
		Channel:    "telegram",
		SilencedBy: "U0123ABC",
		SilencedAt: now,
		ExpiresAt:  &expires,
		Reason:     "maintenance window",
		Active:     true,
	}

	assert.Equal(t, 1, s.ID)
	assert.Equal(t, "check", s.Scope)
	assert.Equal(t, "uuid-123", s.Target)
	assert.Equal(t, "telegram", s.Channel)
	assert.Equal(t, "U0123ABC", s.SilencedBy)
	assert.NotNil(t, s.ExpiresAt)
	assert.Equal(t, expires, *s.ExpiresAt)
	assert.Equal(t, "maintenance window", s.Reason)
	assert.True(t, s.Active)
}

func TestAlertSilence_ProjectScope(t *testing.T) {
	s := AlertSilence{
		Scope:   "project",
		Target:  "my-service",
		Channel: "",
		Active:  true,
	}
	assert.Equal(t, "project", s.Scope)
	assert.Empty(t, s.Channel, "empty channel means all channels")
	assert.True(t, s.Active)
}

func TestAlertSilence_NoExpiry(t *testing.T) {
	s := AlertSilence{
		ID:        3,
		Scope:     "check",
		Target:    "uuid-456",
		ExpiresAt: nil,
		Active:    true,
	}
	assert.Nil(t, s.ExpiresAt)
	assert.True(t, s.Active)
}
