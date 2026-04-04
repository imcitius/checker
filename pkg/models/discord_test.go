package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDiscordAlertThread_ZeroValue(t *testing.T) {
	var d DiscordAlertThread
	assert.Equal(t, 0, d.ID)
	assert.Empty(t, d.CheckUUID)
	assert.Empty(t, d.ChannelID)
	assert.Empty(t, d.MessageID)
	assert.Empty(t, d.ThreadID)
	assert.False(t, d.IsResolved)
	assert.True(t, d.CreatedAt.IsZero())
	assert.Nil(t, d.ResolvedAt)
}

func TestDiscordAlertThread_WithFields(t *testing.T) {
	now := time.Now()
	resolved := now.Add(10 * time.Minute)
	d := DiscordAlertThread{
		ID:         1,
		CheckUUID:  "uuid-discord-1",
		ChannelID:  "123456789",
		MessageID:  "987654321",
		ThreadID:   "111222333",
		IsResolved: true,
		CreatedAt:  now,
		ResolvedAt: &resolved,
	}

	assert.Equal(t, 1, d.ID)
	assert.Equal(t, "uuid-discord-1", d.CheckUUID)
	assert.Equal(t, "123456789", d.ChannelID)
	assert.Equal(t, "987654321", d.MessageID)
	assert.Equal(t, "111222333", d.ThreadID)
	assert.True(t, d.IsResolved)
	assert.NotNil(t, d.ResolvedAt)
	assert.Equal(t, resolved, *d.ResolvedAt)
}

func TestDiscordAlertThread_UnresolvedThread(t *testing.T) {
	d := DiscordAlertThread{
		ID:         2,
		CheckUUID:  "uuid-discord-2",
		ChannelID:  "444555666",
		MessageID:  "777888999",
		ThreadID:   "000111222",
		IsResolved: false,
		CreatedAt:  time.Now(),
		ResolvedAt: nil,
	}
	assert.False(t, d.IsResolved)
	assert.Nil(t, d.ResolvedAt)
}
