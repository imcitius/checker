// SPDX-License-Identifier: BUSL-1.1

package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEdgeConfigSync_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second).UTC()
	msg := EdgeConfigSync{
		Type: "config_sync",
		Checks: []CheckDefinitionViewModel{
			{UUID: "uuid-1", Name: "HTTP Check", Type: "http"},
		},
		ServerTime: now,
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var got EdgeConfigSync
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Equal(t, "config_sync", got.Type)
	assert.Equal(t, now, got.ServerTime)
	assert.Len(t, got.Checks, 1)
	assert.Equal(t, "uuid-1", got.Checks[0].UUID)
	assert.Equal(t, "HTTP Check", got.Checks[0].Name)
}

func TestEdgeConfigPatch_AddAction_JSONRoundTrip(t *testing.T) {
	check := &CheckDefinitionViewModel{UUID: "uuid-2", Name: "TCP Check", Type: "tcp"}
	msg := EdgeConfigPatch{
		Type:   "config_patch",
		Action: "add",
		Check:  check,
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var got EdgeConfigPatch
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Equal(t, "config_patch", got.Type)
	assert.Equal(t, "add", got.Action)
	require.NotNil(t, got.Check)
	assert.Equal(t, "uuid-2", got.Check.UUID)
}

func TestEdgeConfigPatch_DeleteAction_JSONRoundTrip(t *testing.T) {
	msg := EdgeConfigPatch{
		Type:   "config_patch",
		Action: "delete",
		UUID:   "uuid-to-delete",
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var got EdgeConfigPatch
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Equal(t, "config_patch", got.Type)
	assert.Equal(t, "delete", got.Action)
	assert.Equal(t, "uuid-to-delete", got.UUID)
	assert.Nil(t, got.Check)
}

func TestEdgeResult_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second).UTC()
	msg := EdgeResult{
		Type:      "result",
		CheckUUID: "uuid-3",
		IsHealthy: true,
		Message:   "200 OK",
		Duration:  250 * time.Millisecond,
		Timestamp: now,
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var got EdgeResult
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Equal(t, "result", got.Type)
	assert.Equal(t, "uuid-3", got.CheckUUID)
	assert.True(t, got.IsHealthy)
	assert.Equal(t, "200 OK", got.Message)
	assert.Equal(t, 250*time.Millisecond, got.Duration)
	assert.Equal(t, now, got.Timestamp)
}

func TestEdgeHeartbeat_JSONRoundTrip(t *testing.T) {
	msg := EdgeHeartbeat{
		Type:          "heartbeat",
		Version:       "1.2.3",
		Region:        "eu-west-1",
		WorkerCount:   4,
		ActiveChecks:  42,
		UptimeSeconds: 86400,
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var got EdgeHeartbeat
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Equal(t, "heartbeat", got.Type)
	assert.Equal(t, "1.2.3", got.Version)
	assert.Equal(t, "eu-west-1", got.Region)
	assert.Equal(t, 4, got.WorkerCount)
	assert.Equal(t, 42, got.ActiveChecks)
	assert.Equal(t, int64(86400), got.UptimeSeconds)
}

func TestEdgeMessage_JSONRoundTrip(t *testing.T) {
	msg := EdgeMessage{Type: "ping"}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var got EdgeMessage
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Equal(t, "ping", got.Type)
}

func TestEdgeConfigPatch_OmitsCheckWhenNil(t *testing.T) {
	msg := EdgeConfigPatch{
		Type:   "config_patch",
		Action: "delete",
		UUID:   "uuid-x",
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	_, exists := raw["check"]
	assert.False(t, exists, "check field should be omitted when nil")
}

func TestEdgeConfigPatch_OmitsUUIDWhenEmpty(t *testing.T) {
	check := &CheckDefinitionViewModel{UUID: "uuid-y", Name: "check"}
	msg := EdgeConfigPatch{
		Type:   "config_patch",
		Action: "add",
		Check:  check,
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	_, exists := raw["uuid"]
	assert.False(t, exists, "uuid field should be omitted when empty")
}
