package scheduler

import (
	"testing"
	"time"

	"checker/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestShouldSendAlert_HealthyCheck_NeverAlerts(t *testing.T) {
	status := models.CheckStatus{IsHealthy: true}
	assert.False(t, shouldSendAlert(true, 0, status), "healthy check should never trigger alert")
	assert.False(t, shouldSendAlert(false, 0, status), "healthy check should never trigger alert")
	assert.False(t, shouldSendAlert(true, 5*time.Minute, status), "healthy check should never trigger alert even with re-alert interval")
}

func TestShouldSendAlert_StateTransition_HealthyToUnhealthy(t *testing.T) {
	status := models.CheckStatus{IsHealthy: false}

	// Transition from healthy → unhealthy should always alert
	assert.True(t, shouldSendAlert(true, 0, status),
		"healthy→unhealthy transition should trigger alert")
}

func TestShouldSendAlert_OngoingFailure_NoReAlertInterval_NoAlert(t *testing.T) {
	status := models.CheckStatus{
		IsHealthy:     false,
		LastAlertSent: time.Now().Add(-10 * time.Minute),
	}

	// Ongoing failure (unhealthy → unhealthy) without ReAlertInterval should NOT alert
	assert.False(t, shouldSendAlert(false, 0, status),
		"ongoing failure without ReAlertInterval should NOT trigger alert")
}

func TestShouldSendAlert_OngoingFailure_WithReAlertInterval_Elapsed(t *testing.T) {
	status := models.CheckStatus{
		IsHealthy:     false,
		LastAlertSent: time.Now().Add(-10 * time.Minute),
	}

	// ReAlertInterval of 5 minutes, last alert was 10 minutes ago → should re-alert
	assert.True(t, shouldSendAlert(false, 5*time.Minute, status),
		"should re-alert when ReAlertInterval has elapsed")
}

func TestShouldSendAlert_OngoingFailure_WithReAlertInterval_NotElapsed(t *testing.T) {
	status := models.CheckStatus{
		IsHealthy:     false,
		LastAlertSent: time.Now().Add(-2 * time.Minute),
	}

	// ReAlertInterval of 5 minutes, last alert was 2 minutes ago → should NOT re-alert
	assert.False(t, shouldSendAlert(false, 5*time.Minute, status),
		"should NOT re-alert when ReAlertInterval has not elapsed")
}

func TestShouldSendAlert_OngoingFailure_NoLastAlert_WithReAlertInterval(t *testing.T) {
	status := models.CheckStatus{
		IsHealthy: false,
		// LastAlertSent is zero value
	}

	// Ongoing failure, no previous alert ever sent, ReAlertInterval set → should alert
	assert.True(t, shouldSendAlert(false, 5*time.Minute, status),
		"should alert if no previous alert was ever sent and ReAlertInterval is set")
}

// Integration-style scenario tests

func TestScenario_CheckGoesDown_StaysDown_OneAlertOnly(t *testing.T) {
	// Simulate: check goes DOWN once → one alert → stays DOWN → no more alerts (no ReAlertInterval)

	// Cycle 1: healthy → unhealthy (should alert)
	status := models.CheckStatus{IsHealthy: false}
	assert.True(t, shouldSendAlert(true, 0, status),
		"cycle 1: first DOWN should trigger alert")

	// Record that alert was sent
	status.LastAlertSent = time.Now()

	// Cycle 2: unhealthy → unhealthy (should NOT alert, no ReAlertInterval)
	assert.False(t, shouldSendAlert(false, 0, status),
		"cycle 2: ongoing DOWN without ReAlertInterval should NOT trigger alert")

	// Cycle 3: unhealthy → unhealthy (should NOT alert)
	assert.False(t, shouldSendAlert(false, 0, status),
		"cycle 3: still DOWN, still no alert")

	// Cycle 4: unhealthy → unhealthy (should NOT alert)
	assert.False(t, shouldSendAlert(false, 0, status),
		"cycle 4: still DOWN, still no alert")
}

func TestScenario_CheckRecovers(t *testing.T) {
	// The recovery alert is handled outside shouldSendAlert (in executeCheck),
	// but shouldSendAlert should return false for healthy checks.
	status := models.CheckStatus{IsHealthy: true}
	assert.False(t, shouldSendAlert(false, 0, status),
		"recovery check should not trigger DOWN alert")
}

func TestScenario_ReAlertInterval_5m(t *testing.T) {
	reAlertInterval := 5 * time.Minute

	// Cycle 1: healthy → unhealthy (should alert)
	status := models.CheckStatus{IsHealthy: false}
	assert.True(t, shouldSendAlert(true, reAlertInterval, status),
		"cycle 1: first DOWN should trigger alert")

	// Record alert sent 6 minutes ago to simulate passage of time
	status.LastAlertSent = time.Now().Add(-6 * time.Minute)

	// Cycle 2: unhealthy → unhealthy, 6 minutes later (should re-alert)
	assert.True(t, shouldSendAlert(false, reAlertInterval, status),
		"cycle 2: should re-alert after ReAlertInterval elapsed")

	// Record alert sent just now
	status.LastAlertSent = time.Now()

	// Cycle 3: unhealthy → unhealthy, right after alert (should NOT re-alert)
	assert.False(t, shouldSendAlert(false, reAlertInterval, status),
		"cycle 3: should NOT re-alert before ReAlertInterval elapses")
}

func TestScenario_MultipleCyclesDown_OnlyOneAlert(t *testing.T) {
	// Test that multiple consecutive DOWN check cycles produce only one alert
	// when ReAlertInterval is not set

	status := models.CheckStatus{IsHealthy: false}

	// First cycle: healthy → unhealthy
	result1 := shouldSendAlert(true, 0, status)
	assert.True(t, result1, "first DOWN should alert")

	status.LastAlertSent = time.Now()

	// Simulate 10 more check cycles, all unhealthy → unhealthy
	alertCount := 0
	if result1 {
		alertCount++
	}
	for i := 0; i < 10; i++ {
		if shouldSendAlert(false, 0, status) {
			alertCount++
		}
	}

	assert.Equal(t, 1, alertCount, "should have exactly 1 alert for 11 check cycles while DOWN")
}

func TestScenario_DownRecoveryDown(t *testing.T) {
	// Full lifecycle: DOWN → RECOVERY → DOWN again

	status := models.CheckStatus{IsHealthy: false}

	// Step 1: healthy → unhealthy (DOWN alert)
	assert.True(t, shouldSendAlert(true, 0, status),
		"step 1: DOWN alert should fire")
	status.LastAlertSent = time.Now()

	// Step 2: still unhealthy → no alert
	assert.False(t, shouldSendAlert(false, 0, status),
		"step 2: ongoing DOWN, no alert")

	// Step 3: recovery (isHealthy=true) → no DOWN alert
	status.IsHealthy = true
	assert.False(t, shouldSendAlert(false, 0, status),
		"step 3: recovery, no DOWN alert")

	// Step 4: fails again (healthy → unhealthy) → DOWN alert fires again
	status.IsHealthy = false
	assert.True(t, shouldSendAlert(true, 0, status),
		"step 4: new failure after recovery should alert again")
}

// --- configChangedSinceLastAlert tests ---

func TestConfigChangedSinceLastAlert_NoLastAlert(t *testing.T) {
	checkDef := models.CheckDefinition{
		UpdatedAt: time.Now(),
	}
	status := models.CheckStatus{} // LastAlertSent is zero
	assert.False(t, configChangedSinceLastAlert(checkDef, status),
		"should return false when no previous alert was sent")
}

func TestConfigChangedSinceLastAlert_ConfigUpdatedAfterLastAlert(t *testing.T) {
	now := time.Now()
	checkDef := models.CheckDefinition{
		UpdatedAt: now,
	}
	status := models.CheckStatus{
		LastAlertSent: now.Add(-5 * time.Minute),
	}
	assert.True(t, configChangedSinceLastAlert(checkDef, status),
		"should return true when config was updated after last alert")
}

func TestConfigChangedSinceLastAlert_ConfigNotUpdatedSinceLastAlert(t *testing.T) {
	now := time.Now()
	checkDef := models.CheckDefinition{
		UpdatedAt: now.Add(-10 * time.Minute),
	}
	status := models.CheckStatus{
		LastAlertSent: now.Add(-5 * time.Minute),
	}
	assert.False(t, configChangedSinceLastAlert(checkDef, status),
		"should return false when config was not updated since last alert")
}

func TestConfigChangedSinceLastAlert_ZeroUpdatedAt(t *testing.T) {
	checkDef := models.CheckDefinition{} // UpdatedAt is zero
	status := models.CheckStatus{
		LastAlertSent: time.Now().Add(-5 * time.Minute),
	}
	assert.False(t, configChangedSinceLastAlert(checkDef, status),
		"should return false when UpdatedAt is zero")
}

func TestScenario_OngoingFailure_AddNewChannel_AlertsFire(t *testing.T) {
	// Simulate: check is already failing, user adds a new channel → alerts should fire

	now := time.Now()

	// Cycle 1: healthy → unhealthy (initial alert fires via shouldSendAlert)
	status := models.CheckStatus{IsHealthy: false}
	assert.True(t, shouldSendAlert(true, 0, status),
		"cycle 1: first DOWN should trigger alert")

	// Record alert sent
	status.LastAlertSent = now.Add(-10 * time.Minute)

	checkDef := models.CheckDefinition{
		UpdatedAt: now.Add(-20 * time.Minute), // config older than alert
	}

	// Cycle 2: ongoing failure, no config change → no alert
	assert.False(t, shouldSendAlert(false, 0, status),
		"cycle 2: ongoing DOWN without ReAlertInterval should NOT trigger alert")
	assert.False(t, configChangedSinceLastAlert(checkDef, status),
		"cycle 2: config not changed since last alert")

	// User adds a new alert channel → UpdatedAt moves forward
	checkDef.UpdatedAt = now

	// Cycle 3: ongoing failure, config changed → configChangedSinceLastAlert fires
	assert.False(t, shouldSendAlert(false, 0, status),
		"cycle 3: shouldSendAlert still false (no ReAlertInterval)")
	assert.True(t, configChangedSinceLastAlert(checkDef, status),
		"cycle 3: config changed since last alert — new channels should fire")

	// After alerting, LastAlertSent is updated
	status.LastAlertSent = now

	// Cycle 4: ongoing failure, config not changed since last alert → no alert
	assert.False(t, shouldSendAlert(false, 0, status),
		"cycle 4: shouldSendAlert still false")
	assert.False(t, configChangedSinceLastAlert(checkDef, status),
		"cycle 4: config not changed since last alert — no spam")
}
