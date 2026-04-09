// SPDX-License-Identifier: BUSL-1.1

package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppVersion_DefaultEmpty(t *testing.T) {
	// AppVersion is set via ldflags at build time; in tests it should be empty
	assert.Equal(t, "", AppVersion)
}

func TestBuildTime_DefaultEmpty(t *testing.T) {
	// BuildTime is set via ldflags at build time; in tests it should be empty
	assert.Equal(t, "", BuildTime)
}

func TestFrontendVersion_Returns(t *testing.T) {
	// FrontendVersion uses sync.Once so the result is stable across calls.
	// In the test environment it will either read from the embedded spa/.version
	// file or fall back to FRONTEND_VERSION env var or "unknown".
	v := FrontendVersion()
	assert.NotEmpty(t, v, "FrontendVersion should return a non-empty string")

	// Calling it again should return the same value (sync.Once guarantee)
	v2 := FrontendVersion()
	assert.Equal(t, v, v2, "FrontendVersion should be stable across calls")
}
