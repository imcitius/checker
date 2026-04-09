// SPDX-License-Identifier: BUSL-1.1

package checks

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// mockSMTPServer starts a simple SMTP server that responds to EHLO and QUIT.
// Returns the listener address and a cleanup function.
func mockSMTPServer(t *testing.T, startTLS bool) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed
			}
			go handleSMTPConn(conn, startTLS)
		}
	}()

	return ln.Addr().String(), func() { ln.Close() }
}

func handleSMTPConn(conn net.Conn, startTLS bool) {
	defer conn.Close()
	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	// Send greeting
	fmt.Fprintf(writer, "220 mock SMTP server ready\r\n")
	writer.Flush()

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		cmd := strings.ToUpper(line)

		switch {
		case strings.HasPrefix(cmd, "EHLO"):
			fmt.Fprintf(writer, "250-mock.local\r\n")
			if startTLS {
				fmt.Fprintf(writer, "250-STARTTLS\r\n")
			}
			fmt.Fprintf(writer, "250 OK\r\n")
			writer.Flush()
		case strings.HasPrefix(cmd, "QUIT"):
			fmt.Fprintf(writer, "221 Bye\r\n")
			writer.Flush()
			return
		default:
			fmt.Fprintf(writer, "502 Command not implemented\r\n")
			writer.Flush()
		}
	}
}

func TestSMTPCheck_Success(t *testing.T) {
	addr, cleanup := mockSMTPServer(t, false)
	defer cleanup()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	check := &SMTPCheck{
		Host:    host,
		Port:    port,
		Timeout: "5s",
	}

	dur, err := check.Run()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if dur <= 0 {
		t.Error("expected positive duration")
	}
}

func TestSMTPCheck_EmptyHost(t *testing.T) {
	check := &SMTPCheck{
		Host:    "",
		Port:    25,
		Timeout: "5s",
	}

	_, err := check.Run()
	if err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestSMTPCheck_EmptyPort(t *testing.T) {
	check := &SMTPCheck{
		Host:    "127.0.0.1",
		Port:    0,
		Timeout: "5s",
	}

	_, err := check.Run()
	if err == nil {
		t.Fatal("expected error for empty port")
	}
}

func TestSMTPCheck_ConnectionRefused(t *testing.T) {
	// Use a port that's not listening
	check := &SMTPCheck{
		Host:    "127.0.0.1",
		Port:    1, // likely not listening
		Timeout: "1s",
	}

	_, err := check.Run()
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestSMTPCheck_InvalidTimeout(t *testing.T) {
	check := &SMTPCheck{
		Host:    "127.0.0.1",
		Port:    25,
		Timeout: "invalid",
	}

	_, err := check.Run()
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

func TestSMTPCheck_Timeout(t *testing.T) {
	// Create a server that accepts but doesn't respond
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		// Hold connection open without responding
		time.Sleep(5 * time.Second)
		conn.Close()
	}()

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	check := &SMTPCheck{
		Host:    host,
		Port:    port,
		Timeout: "500ms",
	}

	_, err = check.Run()
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
