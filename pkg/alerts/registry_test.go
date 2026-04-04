package alerts

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubAlerter is a minimal Alerter for testing the registry.
type stubAlerter struct {
	channelType string
}

func (s *stubAlerter) SendAlert(_ AlertPayload) error    { return nil }
func (s *stubAlerter) SendRecovery(_ RecoveryPayload) error { return nil }
func (s *stubAlerter) Type() string                       { return s.channelType }

func TestRegisterAlerter_And_NewAlerter(t *testing.T) {
	const testType = "test_channel_register"
	RegisterAlerter(testType, func(config json.RawMessage) (Alerter, error) {
		return &stubAlerter{channelType: testType}, nil
	})
	defer delete(registry, testType)

	a, err := NewAlerter(testType, json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.Equal(t, testType, a.Type())
}

func TestNewAlerter_UnknownType(t *testing.T) {
	_, err := NewAlerter("nonexistent_channel_xyz", json.RawMessage(`{}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown alert channel type")
}

func TestIsRegisteredType_Registered(t *testing.T) {
	const testType = "test_channel_isreg"
	RegisterAlerter(testType, func(config json.RawMessage) (Alerter, error) {
		return &stubAlerter{channelType: testType}, nil
	})
	defer delete(registry, testType)

	assert.True(t, IsRegisteredType(testType))
}

func TestIsRegisteredType_NotRegistered(t *testing.T) {
	assert.False(t, IsRegisteredType("definitely_not_registered_xyz"))
}

func TestRegisterAlerter_OverwritesPrevious(t *testing.T) {
	const testType = "test_channel_overwrite"
	RegisterAlerter(testType, func(config json.RawMessage) (Alerter, error) {
		return &stubAlerter{channelType: "first"}, nil
	})
	RegisterAlerter(testType, func(config json.RawMessage) (Alerter, error) {
		return &stubAlerter{channelType: "second"}, nil
	})
	defer delete(registry, testType)

	a, err := NewAlerter(testType, json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.Equal(t, "second", a.Type())
}

func TestNewAlerter_FactoryError(t *testing.T) {
	const testType = "test_channel_factory_err"
	RegisterAlerter(testType, func(config json.RawMessage) (Alerter, error) {
		return nil, fmt.Errorf("factory failed")
	})
	defer delete(registry, testType)

	_, err := NewAlerter(testType, json.RawMessage(`{}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "factory failed")
}

func TestBuiltInTypesRegistered(t *testing.T) {
	// Verify that the init() registrations from channel files have run.
	builtIn := []string{"telegram", "slack", "discord", "pagerduty", "opsgenie", "teams", "email", "ntfy"}
	for _, ct := range builtIn {
		t.Run(ct, func(t *testing.T) {
			assert.True(t, IsRegisteredType(ct), "expected %q to be registered", ct)
		})
	}
}
