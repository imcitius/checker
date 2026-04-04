package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEscalationStep_JSONRoundTrip(t *testing.T) {
	step := EscalationStep{
		Channel:  "telegram",
		DelayMin: 5,
	}

	data, err := json.Marshal(step)
	assert.NoError(t, err)

	var got EscalationStep
	err = json.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Equal(t, step.Channel, got.Channel)
	assert.Equal(t, step.DelayMin, got.DelayMin)
}

func TestEscalationStep_ZeroValue(t *testing.T) {
	var s EscalationStep
	assert.Empty(t, s.Channel)
	assert.Equal(t, 0, s.DelayMin)
}

func TestEscalationPolicy_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	policy := EscalationPolicy{
		ID:   1,
		Name: "critical-path",
		Steps: []EscalationStep{
			{Channel: "telegram", DelayMin: 0},
			{Channel: "slack", DelayMin: 5},
			{Channel: "pagerduty", DelayMin: 15},
		},
		CreatedAt: now,
	}

	data, err := json.Marshal(policy)
	assert.NoError(t, err)

	var got EscalationPolicy
	err = json.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Equal(t, policy.ID, got.ID)
	assert.Equal(t, policy.Name, got.Name)
	assert.Len(t, got.Steps, 3)
	assert.Equal(t, "telegram", got.Steps[0].Channel)
	assert.Equal(t, 0, got.Steps[0].DelayMin)
	assert.Equal(t, "pagerduty", got.Steps[2].Channel)
	assert.Equal(t, 15, got.Steps[2].DelayMin)
}

func TestEscalationPolicy_EmptySteps(t *testing.T) {
	policy := EscalationPolicy{
		ID:    2,
		Name:  "empty-policy",
		Steps: nil,
	}
	assert.Nil(t, policy.Steps)
	assert.Equal(t, "empty-policy", policy.Name)
}

func TestEscalationPolicy_ZeroValue(t *testing.T) {
	var p EscalationPolicy
	assert.Equal(t, 0, p.ID)
	assert.Empty(t, p.Name)
	assert.Nil(t, p.Steps)
	assert.True(t, p.CreatedAt.IsZero())
}

func TestEscalationNotification_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	n := EscalationNotification{
		ID:         10,
		CheckUUID:  "uuid-abc",
		PolicyName: "critical-path",
		StepIndex:  1,
		NotifiedAt: now,
	}

	data, err := json.Marshal(n)
	assert.NoError(t, err)

	var got EscalationNotification
	err = json.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Equal(t, n.ID, got.ID)
	assert.Equal(t, n.CheckUUID, got.CheckUUID)
	assert.Equal(t, n.PolicyName, got.PolicyName)
	assert.Equal(t, n.StepIndex, got.StepIndex)
}

func TestEscalationNotification_ZeroValue(t *testing.T) {
	var n EscalationNotification
	assert.Equal(t, 0, n.ID)
	assert.Empty(t, n.CheckUUID)
	assert.Empty(t, n.PolicyName)
	assert.Equal(t, 0, n.StepIndex)
	assert.True(t, n.NotifiedAt.IsZero())
}
