package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAlertEvent_ZeroValue(t *testing.T) {
	var e AlertEvent
	assert.Equal(t, 0, e.ID)
	assert.Empty(t, e.CheckUUID)
	assert.Empty(t, e.CheckName)
	assert.Empty(t, e.Project)
	assert.Empty(t, e.GroupName)
	assert.Empty(t, e.CheckType)
	assert.Empty(t, e.Message)
	assert.Empty(t, e.AlertType)
	assert.True(t, e.CreatedAt.IsZero())
	assert.Nil(t, e.ResolvedAt)
	assert.False(t, e.IsResolved)
}

func TestAlertEvent_WithFields(t *testing.T) {
	now := time.Now()
	resolved := now.Add(5 * time.Minute)
	e := AlertEvent{
		ID:         42,
		CheckUUID:  "uuid-123",
		CheckName:  "API Health",
		Project:    "my-service",
		GroupName:  "prod",
		CheckType:  "http",
		Message:    "timeout after 10s",
		AlertType:  "critical",
		CreatedAt:  now,
		ResolvedAt: &resolved,
		IsResolved: true,
	}

	assert.Equal(t, 42, e.ID)
	assert.Equal(t, "uuid-123", e.CheckUUID)
	assert.Equal(t, "API Health", e.CheckName)
	assert.Equal(t, "my-service", e.Project)
	assert.Equal(t, "prod", e.GroupName)
	assert.Equal(t, "http", e.CheckType)
	assert.Equal(t, "timeout after 10s", e.Message)
	assert.Equal(t, "critical", e.AlertType)
	assert.Equal(t, now, e.CreatedAt)
	assert.NotNil(t, e.ResolvedAt)
	assert.Equal(t, resolved, *e.ResolvedAt)
	assert.True(t, e.IsResolved)
}

func TestAlertEvent_NilResolvedAt(t *testing.T) {
	e := AlertEvent{
		ID:         1,
		IsResolved: false,
		ResolvedAt: nil,
	}
	assert.Nil(t, e.ResolvedAt)
	assert.False(t, e.IsResolved)
}

func TestAlertHistoryFilters_ZeroValue(t *testing.T) {
	var f AlertHistoryFilters
	assert.Empty(t, f.Project)
	assert.Empty(t, f.CheckUUID)
	assert.Nil(t, f.IsResolved)
}

func TestAlertHistoryFilters_WithFilters(t *testing.T) {
	resolved := true
	f := AlertHistoryFilters{
		Project:    "my-project",
		CheckUUID:  "uuid-456",
		IsResolved: &resolved,
	}

	assert.Equal(t, "my-project", f.Project)
	assert.Equal(t, "uuid-456", f.CheckUUID)
	assert.NotNil(t, f.IsResolved)
	assert.True(t, *f.IsResolved)
}

func TestAlertHistoryFilters_IsResolvedFalse(t *testing.T) {
	resolved := false
	f := AlertHistoryFilters{
		IsResolved: &resolved,
	}
	assert.NotNil(t, f.IsResolved)
	assert.False(t, *f.IsResolved)
}
