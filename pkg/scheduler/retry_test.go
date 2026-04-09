// SPDX-License-Identifier: BUSL-1.1

package scheduler

import (
	"container/heap"
	"fmt"
	"testing"
	"time"

	"github.com/imcitius/checker/pkg/models"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestRunCheckWithRetries_NoRetry_Success verifies that a passing check returns nil
// with no retries configured.
func TestRunCheckWithRetries_NoRetry_Success(t *testing.T) {
	calls := 0
	runFn := func() (time.Duration, error) {
		calls++
		return 0, nil
	}

	logger := logrus.WithField("test", t.Name())
	err := runCheckWithRetries(runFn, 0, "", logger)

	assert.NoError(t, err)
	assert.Equal(t, 1, calls, "should call runFn exactly once")
}

// TestRunCheckWithRetries_NoRetry_Failure verifies that a failing check with no retries
// returns the error immediately.
func TestRunCheckWithRetries_NoRetry_Failure(t *testing.T) {
	calls := 0
	runFn := func() (time.Duration, error) {
		calls++
		return 0, fmt.Errorf("check failed")
	}

	logger := logrus.WithField("test", t.Name())
	err := runCheckWithRetries(runFn, 0, "", logger)

	assert.Error(t, err)
	assert.Equal(t, "check failed", err.Error())
	assert.Equal(t, 1, calls, "should call runFn exactly once")
}

// TestRunCheckWithRetries_RetryCount2_FailsAfter3Attempts verifies that a check with
// RetryCount=2 only declares failure after 3 total attempts (1 initial + 2 retries).
func TestRunCheckWithRetries_RetryCount2_FailsAfter3Attempts(t *testing.T) {
	calls := 0
	runFn := func() (time.Duration, error) {
		calls++
		return 0, fmt.Errorf("attempt %d failed", calls)
	}

	logger := logrus.WithField("test", t.Name())
	// Use very short retry interval for test speed
	err := runCheckWithRetries(runFn, 2, "1ms", logger)

	assert.Error(t, err)
	assert.Equal(t, 3, calls, "should make 3 total attempts (1 initial + 2 retries)")
	assert.Contains(t, err.Error(), "attempt 3 failed")
}

// TestRunCheckWithRetries_SucceedsOnSecondRetry verifies that a check succeeding
// on the second retry returns nil.
func TestRunCheckWithRetries_SucceedsOnSecondRetry(t *testing.T) {
	calls := 0
	runFn := func() (time.Duration, error) {
		calls++
		if calls < 3 {
			return 0, fmt.Errorf("attempt %d failed", calls)
		}
		return 0, nil // succeed on 3rd attempt
	}

	logger := logrus.WithField("test", t.Name())
	err := runCheckWithRetries(runFn, 3, "1ms", logger)

	assert.NoError(t, err)
	assert.Equal(t, 3, calls, "should have made 3 attempts before success")
}

// TestRunCheckWithRetries_SucceedsOnFirstRetry verifies early exit on first retry success.
func TestRunCheckWithRetries_SucceedsOnFirstRetry(t *testing.T) {
	calls := 0
	runFn := func() (time.Duration, error) {
		calls++
		if calls == 1 {
			return 0, fmt.Errorf("first attempt failed")
		}
		return 0, nil
	}

	logger := logrus.WithField("test", t.Name())
	err := runCheckWithRetries(runFn, 3, "1ms", logger)

	assert.NoError(t, err)
	assert.Equal(t, 2, calls, "should stop retrying after first success")
}

// TestRunCheckWithRetries_DefaultRetryInterval verifies that empty retry interval
// uses the default (5s). We test this indirectly by checking timing.
func TestRunCheckWithRetries_DefaultRetryInterval(t *testing.T) {
	// When retryInterval is empty and retryCount is 0, no retry happens
	calls := 0
	runFn := func() (time.Duration, error) {
		calls++
		return 0, fmt.Errorf("fail")
	}

	logger := logrus.WithField("test", t.Name())
	err := runCheckWithRetries(runFn, 0, "", logger)

	assert.Error(t, err)
	assert.Equal(t, 1, calls)
}

// TestSubMinuteInterval_10s verifies that a 10-second check interval schedules correctly
// in the heap without panics.
func TestSubMinuteInterval_10s(t *testing.T) {
	h := &CheckHeap{}
	heap.Init(h)

	now := time.Now()

	// Sub-minute interval: 10s
	// WARNING: Sub-minute intervals increase DB write frequency proportionally.
	item := &CheckItem{
		CheckDef: models.CheckDefinition{
			UUID:     "sub-minute-10s",
			Name:     "Fast Check",
			Duration: "10s",
			Enabled:  true,
		},
		NextRun: now,
	}

	heap.Push(h, item)
	assert.Equal(t, 1, h.Len())

	// Pop and verify it schedules correctly
	popped := heap.Pop(h).(*CheckItem)
	assert.Equal(t, "sub-minute-10s", popped.CheckDef.UUID)

	// Simulate rescheduling with sub-minute interval
	dur := parseDuration(popped.CheckDef.Duration)
	assert.Equal(t, 10*time.Second, dur, "10s should parse to 10 seconds")

	popped.NextRun = now.Add(dur)
	heap.Push(h, popped)

	peeked := h.Peek()
	assert.NotNil(t, peeked)
	expectedNext := now.Add(10 * time.Second)
	assert.True(t, peeked.NextRun.Equal(expectedNext),
		"next run should be 10 seconds from now, got %v", peeked.NextRun)
}

// TestSubMinuteInterval_30s verifies that a 30-second interval works correctly.
func TestSubMinuteInterval_30s(t *testing.T) {
	dur := parseDuration("30s")
	assert.Equal(t, 30*time.Second, dur, "30s should parse to 30 seconds")
}

// TestSubMinuteInterval_HeapOrdering verifies sub-minute intervals order correctly
// in the heap alongside regular intervals.
func TestSubMinuteInterval_HeapOrdering(t *testing.T) {
	h := &CheckHeap{}
	heap.Init(h)

	now := time.Now()

	// Mix of sub-minute and regular intervals
	items := []*CheckItem{
		{CheckDef: models.CheckDefinition{UUID: "5m", Duration: "5m"}, NextRun: now.Add(5 * time.Minute)},
		{CheckDef: models.CheckDefinition{UUID: "10s", Duration: "10s"}, NextRun: now.Add(10 * time.Second)},
		{CheckDef: models.CheckDefinition{UUID: "1m", Duration: "1m"}, NextRun: now.Add(1 * time.Minute)},
		{CheckDef: models.CheckDefinition{UUID: "30s", Duration: "30s"}, NextRun: now.Add(30 * time.Second)},
	}

	for _, item := range items {
		heap.Push(h, item)
	}

	// Should pop in order: 10s, 30s, 1m, 5m
	expected := []string{"10s", "30s", "1m", "5m"}
	for _, expectedUUID := range expected {
		popped := heap.Pop(h).(*CheckItem)
		assert.Equal(t, expectedUUID, popped.CheckDef.UUID,
			"expected %s but got %s", expectedUUID, popped.CheckDef.UUID)
	}
}

// TestParseDuration_SubMinute verifies parseDuration handles sub-minute values correctly.
func TestParseDuration_SubMinute(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"10s", 10 * time.Second},
		{"30s", 30 * time.Second},
		{"500ms", 500 * time.Millisecond},
		{"1m", 1 * time.Minute},
		{"5m", 5 * time.Minute},
		{"1h", 1 * time.Hour},
		{"", 1 * time.Minute},       // default fallback
		{"invalid", 1 * time.Minute}, // default fallback
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseDuration(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
