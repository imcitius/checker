// SPDX-License-Identifier: BUSL-1.1

package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCheckResult_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	evalAt := now.Add(time.Minute)
	cr := CheckResult{
		ID:          1,
		CheckUUID:   "uuid-abc",
		Region:      "us-east-1",
		IsHealthy:   true,
		Message:     "200 OK",
		CreatedAt:   now,
		CycleKey:    now.Truncate(30 * time.Second),
		EvaluatedAt: &evalAt,
	}

	data, err := json.Marshal(cr)
	assert.NoError(t, err)

	var got CheckResult
	err = json.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Equal(t, cr.ID, got.ID)
	assert.Equal(t, cr.CheckUUID, got.CheckUUID)
	assert.Equal(t, cr.Region, got.Region)
	assert.Equal(t, cr.IsHealthy, got.IsHealthy)
	assert.Equal(t, cr.Message, got.Message)
	assert.NotNil(t, got.EvaluatedAt)
}

func TestCheckResult_ZeroValue(t *testing.T) {
	var cr CheckResult
	assert.Equal(t, int64(0), cr.ID)
	assert.Empty(t, cr.CheckUUID)
	assert.Empty(t, cr.Region)
	assert.False(t, cr.IsHealthy)
	assert.Empty(t, cr.Message)
	assert.True(t, cr.CreatedAt.IsZero())
	assert.True(t, cr.CycleKey.IsZero())
	assert.Nil(t, cr.EvaluatedAt)
}

func TestCheckResult_NilEvaluatedAt(t *testing.T) {
	cr := CheckResult{
		ID:        1,
		CheckUUID: "uuid-123",
		Region:    "eu-west-1",
		IsHealthy: false,
		Message:   "connection refused",
	}

	data, err := json.Marshal(cr)
	assert.NoError(t, err)

	var got CheckResult
	err = json.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Nil(t, got.EvaluatedAt)
}

func TestCheckResult_UnhealthyResult(t *testing.T) {
	cr := CheckResult{
		ID:        2,
		CheckUUID: "uuid-fail",
		Region:    "ap-south-1",
		IsHealthy: false,
		Message:   "timeout after 10s",
		CreatedAt: time.Now(),
		CycleKey:  time.Now().Truncate(time.Minute),
	}
	assert.False(t, cr.IsHealthy)
	assert.Equal(t, "timeout after 10s", cr.Message)
}

func TestCheckResult_JSONOmitsEvaluatedAtWhenNil(t *testing.T) {
	cr := CheckResult{
		ID:        1,
		CheckUUID: "uuid-123",
	}
	data, err := json.Marshal(cr)
	assert.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	assert.NoError(t, err)
	_, exists := raw["evaluated_at"]
	assert.False(t, exists, "evaluated_at should be omitted when nil")
}
