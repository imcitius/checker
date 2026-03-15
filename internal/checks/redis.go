package checks

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	ErrRedisAuth   = "redis AUTH error: %s"
	ErrRedisSelect = "redis SELECT error: %s"
	ErrRedisPing   = "redis PING error: %s"
)

// RedisCheck represents a Redis health check using raw RESP protocol.
type RedisCheck struct {
	Host     string
	Port     int
	Timeout  string
	Password string
	DB       int
	Logger   *logrus.Entry
}

// Run executes the Redis health check via raw TCP + RESP protocol.
func (check *RedisCheck) Run() (time.Duration, error) {
	start := time.Now()

	// Ensure logger is initialized
	if check.Logger == nil {
		check.Logger = logrus.WithField("check", "redis")
	}

	errorHeader := fmt.Sprintf("Redis check error for host %s: ", check.Host)

	if check.Host == "" {
		return time.Since(start), errors.New(errorHeader + ErrEmptyHost)
	}

	port := check.Port
	if port == 0 {
		port = 6379
	}

	// Parse timeout
	if check.Timeout == "" {
		check.Timeout = "5s"
	}
	timeout, err := time.ParseDuration(check.Timeout)
	if err != nil {
		return time.Since(start), fmt.Errorf(errorHeader+"invalid timeout: %v", err)
	}
	if timeout <= 0 {
		return time.Since(start), errors.New(errorHeader + "timeout must be positive")
	}

	// Dial TCP
	hostPort := net.JoinHostPort(check.Host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", hostPort, timeout)
	if err != nil {
		check.Logger.WithError(err).Debugf("Redis TCP dial failed: %s", hostPort)
		return time.Since(start), err
	}
	defer conn.Close()

	// Set read/write deadline
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return time.Since(start), fmt.Errorf(errorHeader+"set deadline: %v", err)
	}

	reader := bufio.NewReader(conn)

	// AUTH if password is set
	if check.Password != "" {
		if err := respCommand(conn, reader, fmt.Sprintf("AUTH %s", check.Password), "+OK"); err != nil {
			return time.Since(start), fmt.Errorf(ErrRedisAuth, err)
		}
		check.Logger.Debug("Redis AUTH successful")
	}

	// SELECT db if not default
	if check.DB != 0 {
		if err := respCommand(conn, reader, fmt.Sprintf("SELECT %d", check.DB), "+OK"); err != nil {
			return time.Since(start), fmt.Errorf(ErrRedisSelect, err)
		}
		check.Logger.Debugf("Redis SELECT %d successful", check.DB)
	}

	// PING
	if err := respCommand(conn, reader, "PING", "+PONG"); err != nil {
		return time.Since(start), fmt.Errorf(ErrRedisPing, err)
	}
	check.Logger.Debug("Redis PING/PONG successful")

	return time.Since(start), nil
}

// respCommand sends a RESP inline command and validates the response prefix.
func respCommand(conn net.Conn, reader *bufio.Reader, cmd string, expectPrefix string) error {
	// Send command using inline format
	_, err := fmt.Fprintf(conn, "%s\r\n", cmd)
	if err != nil {
		return fmt.Errorf("send %q: %v", cmd, err)
	}

	// Read response line
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read response for %q: %v", cmd, err)
	}

	line = strings.TrimRight(line, "\r\n")

	// Check for error response
	if strings.HasPrefix(line, "-") {
		return fmt.Errorf("server error for %q: %s", cmd, line[1:])
	}

	// Validate expected response
	if !strings.HasPrefix(line, expectPrefix) {
		return fmt.Errorf("unexpected response for %q: got %q, want prefix %q", cmd, line, expectPrefix)
	}

	return nil
}
