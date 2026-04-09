// SPDX-License-Identifier: BUSL-1.1

package checks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Compile-time interface compliance checks for all Checker implementations.
var (
	_ Checker = (*HTTPCheck)(nil)
	_ Checker = (*TCPCheck)(nil)
	_ Checker = (*SSHCheck)(nil)
	_ Checker = (*ICMPCheck)(nil)
	_ Checker = (*DNSCheck)(nil)
	_ Checker = (*PassiveCheck)(nil)
	_ Checker = (*MySQLCheck)(nil)
	_ Checker = (*MySQLTimeCheck)(nil)
	_ Checker = (*MySQLReplicationCheck)(nil)
	_ Checker = (*PostgreSQLCheck)(nil)
	_ Checker = (*PostgreSQLTimeCheck)(nil)
	_ Checker = (*PostgreSQLReplicationCheck)(nil)
	_ Checker = (*RedisCheck)(nil)
	_ Checker = (*MongoDBCheck)(nil)
	_ Checker = (*DomainExpiryCheck)(nil)
	_ Checker = (*SSLCertCheck)(nil)
	_ Checker = (*SMTPCheck)(nil)
	_ Checker = (*GRPCHealthCheck)(nil)
	_ Checker = (*WebSocketCheck)(nil)
)

func TestCheckerInterfaceRunReturnTypes(t *testing.T) {
	// Verify the Checker interface signature matches expected return types.
	// This is a compile-time check; the test body confirms types at runtime.
	var c Checker = &TCPCheck{Host: "127.0.0.1", Port: 1, Timeout: "1s"}
	_, err := c.Run()
	// Connection will fail but the important thing is the return types are correct.
	assert.Error(t, err)
}

func TestCheckerConstants(t *testing.T) {
	assert.Equal(t, "empty host", ErrEmptyHost)
	assert.Equal(t, "port is empty", ErrEmptyPort)
	assert.Contains(t, ErrICMPError, "icmp error")
	assert.Contains(t, ErrPacketLoss, "icmp error")
	assert.Equal(t, "http client construction error", ErrHTTPClientConstruction)
}

func TestDefaultSSHPort(t *testing.T) {
	assert.Equal(t, 22, DefaultSSHPort)
}

func TestCheckerRunReturnsDuration(t *testing.T) {
	// A quick functional check that Run returns a positive duration on success.
	// Using PassiveCheck with a recent LastPing to avoid external dependencies.
	pc := &PassiveCheck{
		LastPing:    time.Now(),
		Timeout:     "300s",
		ErrorHeader: "test passive check",
	}
	dur, err := pc.Run()
	assert.NoError(t, err)
	assert.True(t, dur >= 0, "duration should be non-negative")
}
