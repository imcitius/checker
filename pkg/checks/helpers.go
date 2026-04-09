// SPDX-License-Identifier: BUSL-1.1

package checks

import (
	"fmt"
	"time"
)

// parseCheckTimeout parses a duration string, returning defaultDur if s is empty.
// This prevents the 'invalid duration ""' error that occurs when timeout is not configured.
func parseCheckTimeout(s string, defaultDur time.Duration) (time.Duration, error) {
	if s == "" {
		return defaultDur, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid timeout %q: %w", s, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("timeout must be positive, got %q", s)
	}
	return d, nil
}
