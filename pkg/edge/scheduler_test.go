package edge

import (
	"context"
	"testing"
	"time"

	"github.com/imcitius/checker/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeDef is a helper that builds a minimal enabled CheckDefinition.
func makeDef(uuid, typ, duration string) models.CheckDefinition {
	return models.CheckDefinition{
		UUID:     uuid,
		Name:     "test-" + uuid,
		Type:     typ,
		Duration: duration,
		Enabled:  true,
	}
}

// TestEdgeScheduler_ReplaceAll verifies that ReplaceAll populates the check map.
func TestEdgeScheduler_ReplaceAll(t *testing.T) {
	results := make(chan CheckResult, 16)
	sched := NewEdgeScheduler(2, results)

	defs := []models.CheckDefinition{
		makeDef("uuid-1", "http", "1m"),
		makeDef("uuid-2", "tcp", "30s"),
	}
	// A disabled check should NOT be included.
	disabled := makeDef("uuid-3", "http", "1m")
	disabled.Enabled = false
	defs = append(defs, disabled)

	sched.ReplaceAll(defs)

	assert.Equal(t, 2, sched.ActiveCount(), "only enabled checks should be scheduled")
}

// TestEdgeScheduler_AddOrUpdate_Add verifies that a new check is added.
func TestEdgeScheduler_AddOrUpdate_Add(t *testing.T) {
	results := make(chan CheckResult, 16)
	sched := NewEdgeScheduler(2, results)

	sched.AddOrUpdate(makeDef("uuid-1", "http", "1m"))
	assert.Equal(t, 1, sched.ActiveCount())

	sched.AddOrUpdate(makeDef("uuid-2", "tcp", "30s"))
	assert.Equal(t, 2, sched.ActiveCount())
}

// TestEdgeScheduler_AddOrUpdate_Update verifies that an existing check is updated.
func TestEdgeScheduler_AddOrUpdate_Update(t *testing.T) {
	results := make(chan CheckResult, 16)
	sched := NewEdgeScheduler(2, results)

	sched.AddOrUpdate(makeDef("uuid-1", "http", "1m"))
	assert.Equal(t, 1, sched.ActiveCount())

	// Update same UUID — count must not change.
	updated := makeDef("uuid-1", "http", "2m")
	sched.AddOrUpdate(updated)
	assert.Equal(t, 1, sched.ActiveCount())

	// Verify the definition was actually updated.
	sched.mu.Lock()
	item := sched.checkMap["uuid-1"]
	sched.mu.Unlock()
	require.NotNil(t, item)
	assert.Equal(t, "2m", item.CheckDef.Duration)
}

// TestEdgeScheduler_AddOrUpdate_Disable verifies that disabling a check removes it.
func TestEdgeScheduler_AddOrUpdate_Disable(t *testing.T) {
	results := make(chan CheckResult, 16)
	sched := NewEdgeScheduler(2, results)

	sched.AddOrUpdate(makeDef("uuid-1", "http", "1m"))
	assert.Equal(t, 1, sched.ActiveCount())

	disabled := makeDef("uuid-1", "http", "1m")
	disabled.Enabled = false
	sched.AddOrUpdate(disabled)
	assert.Equal(t, 0, sched.ActiveCount())
}

// TestEdgeScheduler_Delete verifies that a check can be removed.
func TestEdgeScheduler_Delete(t *testing.T) {
	results := make(chan CheckResult, 16)
	sched := NewEdgeScheduler(2, results)

	sched.AddOrUpdate(makeDef("uuid-1", "http", "1m"))
	sched.AddOrUpdate(makeDef("uuid-2", "tcp", "30s"))
	assert.Equal(t, 2, sched.ActiveCount())

	sched.Delete("uuid-1")
	assert.Equal(t, 1, sched.ActiveCount())

	// Deleting a non-existent UUID should be a no-op.
	sched.Delete("does-not-exist")
	assert.Equal(t, 1, sched.ActiveCount())
}

// TestEdgeScheduler_ResultProduced verifies that the scheduler produces results
// for checks that are due immediately. We use a very short duration and a
// passive check (which returns immediately) to avoid network calls in tests.
func TestEdgeScheduler_ResultProduced(t *testing.T) {
	results := make(chan CheckResult, 16)
	sched := NewEdgeScheduler(2, results)

	// Use "passive" type — no network calls, just returns immediately.
	def := models.CheckDefinition{
		UUID:     "passive-1",
		Name:     "passive test",
		Type:     "passive",
		Duration: "1s", // short interval
		Enabled:  true,
		Config:   &models.PassiveCheckConfig{},
	}
	sched.AddOrUpdate(def)

	// Force an immediate dispatch by setting NextRun to the past.
	sched.mu.Lock()
	if item, ok := sched.checkMap["passive-1"]; ok {
		item.NextRun = time.Now().Add(-time.Second)
	}
	sched.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go sched.Run(ctx)

	// Wait for at least one result.
	select {
	case r := <-results:
		// We received a result — UUID should match regardless of health status.
		assert.Equal(t, "passive-1", r.CheckUUID)
	case <-ctx.Done():
		t.Fatal("timed out waiting for check result")
	}
}

// TestEdgeScheduler_ReplaceAll_ClearsExisting verifies that ReplaceAll resets state.
func TestEdgeScheduler_ReplaceAll_ClearsExisting(t *testing.T) {
	results := make(chan CheckResult, 16)
	sched := NewEdgeScheduler(2, results)

	sched.AddOrUpdate(makeDef("uuid-old", "http", "1m"))
	assert.Equal(t, 1, sched.ActiveCount())

	// Replace with a completely different set.
	sched.ReplaceAll([]models.CheckDefinition{
		makeDef("uuid-new-1", "tcp", "30s"),
		makeDef("uuid-new-2", "tcp", "30s"),
	})

	assert.Equal(t, 2, sched.ActiveCount())

	sched.mu.Lock()
	_, oldExists := sched.checkMap["uuid-old"]
	sched.mu.Unlock()
	assert.False(t, oldExists, "old check should have been removed")
}

// TestViewModelToCheckDef_HTTP verifies that HTTP view models are correctly converted.
func TestViewModelToCheckDef_HTTP(t *testing.T) {
	vm := models.CheckDefinitionViewModel{
		UUID:     "test-http",
		Name:     "HTTP Test",
		Type:     "http",
		Enabled:  true,
		Duration: "1m",
		URL:      "https://example.com",
		Timeout:  "10s",
		Code:     []int{200, 201},
	}

	def := viewModelToCheckDef(vm)

	assert.Equal(t, "test-http", def.UUID)
	assert.Equal(t, "http", def.Type)
	require.NotNil(t, def.Config)

	cfg, ok := def.Config.(*models.HTTPCheckConfig)
	require.True(t, ok, "expected *HTTPCheckConfig")
	assert.Equal(t, "https://example.com", cfg.URL)
	assert.Equal(t, "10s", cfg.Timeout)
	assert.Equal(t, []int{200, 201}, cfg.Code)
}

// TestViewModelToCheckDef_TCP verifies TCP conversion.
func TestViewModelToCheckDef_TCP(t *testing.T) {
	vm := models.CheckDefinitionViewModel{
		UUID:    "tcp-1",
		Type:    "tcp",
		Enabled: true,
		Host:    "db.example.com",
		Port:    5432,
		Timeout: "5s",
	}
	def := viewModelToCheckDef(vm)
	cfg, ok := def.Config.(*models.TCPCheckConfig)
	require.True(t, ok)
	assert.Equal(t, "db.example.com", cfg.Host)
	assert.Equal(t, 5432, cfg.Port)
}

// TestViewModelToCheckDef_Unknown verifies graceful handling of unknown types.
func TestViewModelToCheckDef_Unknown(t *testing.T) {
	vm := models.CheckDefinitionViewModel{
		UUID:    "unknown-1",
		Type:    "totally_unknown_type",
		Enabled: true,
	}
	def := viewModelToCheckDef(vm)
	assert.Equal(t, "unknown-1", def.UUID) // just ensure it doesn't panic
	assert.Nil(t, def.Config)
}

// TestParseDuration verifies duration parsing helper.
func TestParseDuration(t *testing.T) {
	assert.Equal(t, 30*time.Second, parseDuration("30s"))
	assert.Equal(t, 5*time.Minute, parseDuration("5m"))
	assert.Equal(t, time.Duration(0), parseDuration(""))
	assert.Equal(t, time.Duration(0), parseDuration("bad"))
}
