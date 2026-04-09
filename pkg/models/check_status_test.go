// SPDX-License-Identifier: BUSL-1.1

package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCheckStatus_JSONRoundTrip(t *testing.T) {
	id := primitive.NewObjectID()
	now := time.Now().Truncate(time.Millisecond)
	cs := CheckStatus{
		ID:            id,
		UUID:          "cs-uuid-1",
		Project:       "my-project",
		CheckGroup:    "prod",
		CheckName:     "API Health",
		CheckType:     "http",
		LastRun:       now,
		IsHealthy:     true,
		Message:       "200 OK",
		IsEnabled:     true,
		LastAlertSent: now.Add(-10 * time.Minute),
		Host:          "api.example.com",
		Periodicity:   "1m",
		URL:           "https://api.example.com/health",
		IsSilenced:    false,
	}

	data, err := json.Marshal(cs)
	assert.NoError(t, err)

	var got CheckStatus
	err = json.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Equal(t, cs.UUID, got.UUID)
	assert.Equal(t, cs.Project, got.Project)
	assert.Equal(t, cs.CheckName, got.CheckName)
	assert.Equal(t, cs.CheckType, got.CheckType)
	assert.Equal(t, cs.IsHealthy, got.IsHealthy)
	assert.Equal(t, cs.IsEnabled, got.IsEnabled)
	assert.Equal(t, cs.IsSilenced, got.IsSilenced)
}

func TestCheckStatus_BSONRoundTrip(t *testing.T) {
	id := primitive.NewObjectID()
	now := time.Now().Truncate(time.Millisecond)
	cs := CheckStatus{
		ID:         id,
		UUID:       "cs-uuid-2",
		Project:    "test-project",
		CheckGroup: "staging",
		CheckName:  "DB Check",
		CheckType:  "pgsql_query",
		LastRun:    now,
		IsHealthy:  false,
		Message:    "connection refused",
		IsEnabled:  true,
		Host:       "db.example.com",
	}

	data, err := bson.Marshal(cs)
	assert.NoError(t, err)

	var got CheckStatus
	err = bson.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Equal(t, cs.UUID, got.UUID)
	assert.Equal(t, cs.Project, got.Project)
	assert.Equal(t, cs.CheckName, got.CheckName)
	assert.Equal(t, cs.IsHealthy, got.IsHealthy)
}

func TestCheckStatus_ZeroValue(t *testing.T) {
	var cs CheckStatus
	assert.True(t, cs.ID.IsZero())
	assert.Empty(t, cs.UUID)
	assert.Empty(t, cs.Project)
	assert.Empty(t, cs.CheckGroup)
	assert.Empty(t, cs.CheckName)
	assert.Empty(t, cs.CheckType)
	assert.True(t, cs.LastRun.IsZero())
	assert.False(t, cs.IsHealthy)
	assert.Empty(t, cs.Message)
	assert.False(t, cs.IsEnabled)
	assert.True(t, cs.LastAlertSent.IsZero())
	assert.Empty(t, cs.Host)
	assert.Empty(t, cs.Periodicity)
	assert.Empty(t, cs.URL)
	assert.False(t, cs.IsSilenced)
}

func TestCheckStatus_SilencedCheck(t *testing.T) {
	cs := CheckStatus{
		UUID:       "silenced-uuid",
		CheckName:  "Noisy Check",
		IsHealthy:  false,
		IsSilenced: true,
		IsEnabled:  true,
	}
	assert.True(t, cs.IsSilenced)
	assert.False(t, cs.IsHealthy)
	assert.True(t, cs.IsEnabled)
}
