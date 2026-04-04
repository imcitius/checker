package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckDefaults_JSONRoundTrip(t *testing.T) {
	d := CheckDefaults{
		RetryCount:       3,
		RetryInterval:    "5s",
		CheckInterval:    "1m",
		Timeouts:         map[string]string{"http": "10s", "tcp": "5s", "dns": "3s"},
		ReAlertInterval:  "30m",
		Severity:         "critical",
		AlertChannels:    []string{"telegram", "slack"},
		EscalationPolicy: "default-policy",
	}

	data, err := json.Marshal(d)
	assert.NoError(t, err)

	var got CheckDefaults
	err = json.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Equal(t, d.RetryCount, got.RetryCount)
	assert.Equal(t, d.RetryInterval, got.RetryInterval)
	assert.Equal(t, d.CheckInterval, got.CheckInterval)
	assert.Equal(t, d.Timeouts, got.Timeouts)
	assert.Equal(t, d.ReAlertInterval, got.ReAlertInterval)
	assert.Equal(t, d.Severity, got.Severity)
	assert.Equal(t, d.AlertChannels, got.AlertChannels)
	assert.Equal(t, d.EscalationPolicy, got.EscalationPolicy)
}

func TestCheckDefaults_ZeroValue(t *testing.T) {
	var d CheckDefaults
	assert.Equal(t, 0, d.RetryCount)
	assert.Empty(t, d.RetryInterval)
	assert.Empty(t, d.CheckInterval)
	assert.Nil(t, d.Timeouts)
	assert.Empty(t, d.ReAlertInterval)
	assert.Empty(t, d.Severity)
	assert.Nil(t, d.AlertChannels)
	assert.Empty(t, d.EscalationPolicy)
}

func TestCheckDefaults_EmptyTimeouts(t *testing.T) {
	d := CheckDefaults{
		Timeouts: map[string]string{},
	}
	assert.NotNil(t, d.Timeouts)
	assert.Empty(t, d.Timeouts)
}

func TestCheckDefaults_NilAlertChannels(t *testing.T) {
	d := CheckDefaults{
		AlertChannels: nil,
	}
	data, err := json.Marshal(d)
	assert.NoError(t, err)

	var got CheckDefaults
	err = json.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Nil(t, got.AlertChannels)
}
