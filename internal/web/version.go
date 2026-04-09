// SPDX-License-Identifier: BUSL-1.1

package web

import (
	"os"
	"strings"
	"sync"
)

// VersionInfo holds build version metadata injected at compile time.
var (
	// Set via ldflags at build time
	AppVersion string
	BuildTime  string

	frontendVersion     string
	frontendVersionOnce sync.Once
)

// FrontendVersion reads the frontend git SHA from the embedded .version file
// in the SPA directory (written during Docker/CI build).
func FrontendVersion() string {
	frontendVersionOnce.Do(func() {
		// Try reading from the embedded SPA filesystem first
		data, err := spaFS.ReadFile("spa/.version")
		if err == nil {
			frontendVersion = strings.TrimSpace(string(data))
			return
		}

		// Fallback: check environment variable
		if v := os.Getenv("FRONTEND_VERSION"); v != "" {
			frontendVersion = v
			return
		}

		frontendVersion = "unknown"
	})
	return frontendVersion
}
