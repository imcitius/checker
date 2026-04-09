// SPDX-License-Identifier: BUSL-1.1

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckViewModel_ZeroValue(t *testing.T) {
	var vm CheckViewModel
	assert.Empty(t, vm.ID)
	assert.Empty(t, vm.Name)
	assert.Empty(t, vm.Project)
	assert.Empty(t, vm.Healthcheck)
	assert.False(t, vm.LastResult)
	assert.Empty(t, vm.LastExec)
	assert.Empty(t, vm.LastPing)
	assert.False(t, vm.Enabled)
	assert.Empty(t, vm.UUID)
	assert.Empty(t, vm.CheckType)
	assert.Empty(t, vm.Message)
	assert.Empty(t, vm.Host)
	assert.Empty(t, vm.Periodicity)
	assert.Empty(t, vm.URL)
	assert.False(t, vm.IsSilenced)
}

func TestCheckViewModel_WithFields(t *testing.T) {
	vm := CheckViewModel{
		ID:          "1",
		Name:        "API Health",
		Project:     "my-service",
		Healthcheck: "prod",
		LastResult:  true,
		LastExec:    "2024-01-01T00:00:00Z",
		LastPing:    "2024-01-01T00:05:00Z",
		Enabled:     true,
		UUID:        "uuid-123",
		CheckType:   "http",
		Message:     "200 OK",
		Host:        "api.example.com",
		Periodicity: "1m",
		URL:         "https://api.example.com/health",
		IsSilenced:  false,
	}

	assert.Equal(t, "1", vm.ID)
	assert.Equal(t, "API Health", vm.Name)
	assert.Equal(t, "my-service", vm.Project)
	assert.Equal(t, "prod", vm.Healthcheck)
	assert.True(t, vm.LastResult)
	assert.Equal(t, "2024-01-01T00:00:00Z", vm.LastExec)
	assert.True(t, vm.Enabled)
	assert.Equal(t, "http", vm.CheckType)
	assert.Equal(t, "https://api.example.com/health", vm.URL)
}

func TestCheckViewModel_SilencedUnhealthy(t *testing.T) {
	vm := CheckViewModel{
		UUID:       "silenced-uuid",
		Name:       "Flaky Check",
		LastResult: false,
		IsSilenced: true,
		Enabled:    true,
	}
	assert.True(t, vm.IsSilenced)
	assert.False(t, vm.LastResult)
	assert.True(t, vm.Enabled)
}

func TestCheckViewModel_DisabledCheck(t *testing.T) {
	vm := CheckViewModel{
		UUID:    "disabled-uuid",
		Enabled: false,
	}
	assert.False(t, vm.Enabled)
}
