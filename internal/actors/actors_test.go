// SPDX-License-Identifier: BUSL-1.1

package actors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLogActorImplementsActor verifies that LogActor satisfies the Actor interface.
func TestLogActorImplementsActor(t *testing.T) {
	var a Actor = &LogActor{Message: "test"}
	assert.NotNil(t, a)
}
