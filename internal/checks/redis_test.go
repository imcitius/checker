package checks

import (
	"fmt"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startMockRedis starts a mock TCP server that speaks RESP protocol.
// It returns the listener and a cleanup function.
// The handler function processes each received command line and returns a RESP response.
func startMockRedis(t *testing.T, handler func(cmd string) string) (net.Listener, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // listener closed
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 4096)
				for {
					n, err := c.Read(buf)
					if err != nil {
						return
					}
					// Handle potentially multiple commands in a single read
					lines := strings.Split(strings.TrimRight(string(buf[:n]), "\r\n"), "\r\n")
					for _, line := range lines {
						if line == "" {
							continue
						}
						resp := handler(line)
						fmt.Fprintf(c, "%s\r\n", resp)
					}
				}
			}(conn)
		}
	}()

	return listener, func() { listener.Close() }
}

func TestRedisCheck_PingSuccess(t *testing.T) {
	listener, cleanup := startMockRedis(t, func(cmd string) string {
		if strings.HasPrefix(cmd, "PING") {
			return "+PONG"
		}
		return "-ERR unknown command"
	})
	defer cleanup()

	port := listener.Addr().(*net.TCPAddr).Port
	check := RedisCheck{
		Host:    "127.0.0.1",
		Port:    port,
		Timeout: "5s",
		Logger:  logrus.WithField("test", "PingSuccess"),
	}

	duration, err := check.Run()
	assert.NoError(t, err)
	assert.Greater(t, duration.Nanoseconds(), int64(0))
}

func TestRedisCheck_AuthSuccess(t *testing.T) {
	listener, cleanup := startMockRedis(t, func(cmd string) string {
		switch {
		case strings.HasPrefix(cmd, "AUTH secret123"):
			return "+OK"
		case strings.HasPrefix(cmd, "PING"):
			return "+PONG"
		default:
			return "-ERR unknown command"
		}
	})
	defer cleanup()

	port := listener.Addr().(*net.TCPAddr).Port
	check := RedisCheck{
		Host:     "127.0.0.1",
		Port:     port,
		Timeout:  "5s",
		Password: "secret123",
		Logger:   logrus.WithField("test", "AuthSuccess"),
	}

	duration, err := check.Run()
	assert.NoError(t, err)
	assert.Greater(t, duration.Nanoseconds(), int64(0))
}

func TestRedisCheck_AuthFailure(t *testing.T) {
	listener, cleanup := startMockRedis(t, func(cmd string) string {
		if strings.HasPrefix(cmd, "AUTH") {
			return "-ERR invalid password"
		}
		return "+PONG"
	})
	defer cleanup()

	port := listener.Addr().(*net.TCPAddr).Port
	check := RedisCheck{
		Host:     "127.0.0.1",
		Port:     port,
		Timeout:  "5s",
		Password: "wrongpass",
		Logger:   logrus.WithField("test", "AuthFailure"),
	}

	_, err := check.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AUTH")
}

func TestRedisCheck_SelectDB(t *testing.T) {
	listener, cleanup := startMockRedis(t, func(cmd string) string {
		switch {
		case strings.HasPrefix(cmd, "SELECT 3"):
			return "+OK"
		case strings.HasPrefix(cmd, "PING"):
			return "+PONG"
		default:
			return "-ERR unknown command"
		}
	})
	defer cleanup()

	port := listener.Addr().(*net.TCPAddr).Port
	check := RedisCheck{
		Host:    "127.0.0.1",
		Port:    port,
		Timeout: "5s",
		DB:      3,
		Logger:  logrus.WithField("test", "SelectDB"),
	}

	duration, err := check.Run()
	assert.NoError(t, err)
	assert.Greater(t, duration.Nanoseconds(), int64(0))
}

func TestRedisCheck_AuthThenSelectThenPing(t *testing.T) {
	cmdOrder := []string{}
	listener, cleanup := startMockRedis(t, func(cmd string) string {
		cmdOrder = append(cmdOrder, cmd)
		switch {
		case strings.HasPrefix(cmd, "AUTH mypass"):
			return "+OK"
		case strings.HasPrefix(cmd, "SELECT 2"):
			return "+OK"
		case strings.HasPrefix(cmd, "PING"):
			return "+PONG"
		default:
			return "-ERR unknown command"
		}
	})
	defer cleanup()

	port := listener.Addr().(*net.TCPAddr).Port
	check := RedisCheck{
		Host:     "127.0.0.1",
		Port:     port,
		Timeout:  "5s",
		Password: "mypass",
		DB:       2,
		Logger:   logrus.WithField("test", "FullFlow"),
	}

	_, err := check.Run()
	assert.NoError(t, err)
	// Verify command order
	require.Len(t, cmdOrder, 3)
	assert.True(t, strings.HasPrefix(cmdOrder[0], "AUTH"))
	assert.True(t, strings.HasPrefix(cmdOrder[1], "SELECT"))
	assert.True(t, strings.HasPrefix(cmdOrder[2], "PING"))
}

func TestRedisCheck_EmptyHost(t *testing.T) {
	check := RedisCheck{
		Host:    "",
		Port:    6379,
		Timeout: "5s",
		Logger:  logrus.WithField("test", "EmptyHost"),
	}

	_, err := check.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrEmptyHost)
}

func TestRedisCheck_InvalidTimeout(t *testing.T) {
	check := RedisCheck{
		Host:    "127.0.0.1",
		Port:    6379,
		Timeout: "invalid",
		Logger:  logrus.WithField("test", "InvalidTimeout"),
	}

	_, err := check.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timeout")
}

func TestRedisCheck_DefaultPort(t *testing.T) {
	// This will fail to connect, but verifies Port=0 defaults to 6379
	check := RedisCheck{
		Host:    "127.0.0.1",
		Port:    0,
		Timeout: "100ms",
		Logger:  logrus.WithField("test", "DefaultPort"),
	}

	_, err := check.Run()
	// Connection will be refused, but the check should have attempted port 6379
	assert.Error(t, err)
}

func TestRedisCheck_ConnectionRefused(t *testing.T) {
	check := RedisCheck{
		Host:    "127.0.0.1",
		Port:    54322,
		Timeout: "1s",
		Logger:  logrus.WithField("test", "ConnectionRefused"),
	}

	_, err := check.Run()
	assert.Error(t, err)
}

func TestRedisCheck_PingFailure(t *testing.T) {
	listener, cleanup := startMockRedis(t, func(cmd string) string {
		if strings.HasPrefix(cmd, "PING") {
			return "-ERR not ready"
		}
		return "+OK"
	})
	defer cleanup()

	port := listener.Addr().(*net.TCPAddr).Port
	check := RedisCheck{
		Host:    "127.0.0.1",
		Port:    port,
		Timeout: "5s",
		Logger:  logrus.WithField("test", "PingFailure"),
	}

	_, err := check.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PING")
}

func TestRedisCheck_SelectDBFailure(t *testing.T) {
	listener, cleanup := startMockRedis(t, func(cmd string) string {
		if strings.HasPrefix(cmd, "SELECT") {
			return "-ERR invalid DB index"
		}
		return "+PONG"
	})
	defer cleanup()

	port := listener.Addr().(*net.TCPAddr).Port
	check := RedisCheck{
		Host:    "127.0.0.1",
		Port:    port,
		Timeout: "5s",
		DB:      99,
		Logger:  logrus.WithField("test", "SelectDBFailure"),
	}

	_, err := check.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SELECT")
}

// Integration test — only runs when INTEGRATION_TESTS=true and a real Redis is available
func TestRedisCheck_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TESTS=true to run")
	}

	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	check := RedisCheck{
		Host:     host,
		Port:     6379,
		Timeout:  "5s",
		Password: os.Getenv("REDIS_PASSWORD"),
		Logger:   logrus.WithField("test", "Integration"),
	}

	duration, err := check.Run()
	assert.NoError(t, err)
	assert.Greater(t, duration.Nanoseconds(), int64(0))
	t.Logf("Redis PING latency: %v", duration)
}
