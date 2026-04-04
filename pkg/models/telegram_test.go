package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTelegramAlertThread_ZeroValue(t *testing.T) {
	var tg TelegramAlertThread
	assert.Equal(t, 0, tg.ID)
	assert.Empty(t, tg.CheckUUID)
	assert.Empty(t, tg.ChatID)
	assert.Equal(t, 0, tg.MessageID)
	assert.False(t, tg.IsResolved)
	assert.True(t, tg.CreatedAt.IsZero())
	assert.Nil(t, tg.ResolvedAt)
}

func TestTelegramAlertThread_WithFields(t *testing.T) {
	now := time.Now()
	resolved := now.Add(15 * time.Minute)
	tg := TelegramAlertThread{
		ID:         1,
		CheckUUID:  "uuid-tg-1",
		ChatID:     "-100123456789",
		MessageID:  42,
		IsResolved: true,
		CreatedAt:  now,
		ResolvedAt: &resolved,
	}

	assert.Equal(t, 1, tg.ID)
	assert.Equal(t, "uuid-tg-1", tg.CheckUUID)
	assert.Equal(t, "-100123456789", tg.ChatID)
	assert.Equal(t, 42, tg.MessageID)
	assert.True(t, tg.IsResolved)
	assert.NotNil(t, tg.ResolvedAt)
	assert.Equal(t, resolved, *tg.ResolvedAt)
}

func TestTelegramAlertThread_UnresolvedThread(t *testing.T) {
	tg := TelegramAlertThread{
		ID:         2,
		CheckUUID:  "uuid-tg-2",
		ChatID:     "-100987654321",
		MessageID:  99,
		IsResolved: false,
		CreatedAt:  time.Now(),
		ResolvedAt: nil,
	}
	assert.False(t, tg.IsResolved)
	assert.Nil(t, tg.ResolvedAt)
}

func TestTelegramAlertThread_ZeroMessageID(t *testing.T) {
	tg := TelegramAlertThread{
		MessageID: 0,
	}
	assert.Equal(t, 0, tg.MessageID)
}
