package checks

import (
    "fmt"
    "net"
    "time"
)

// TCPCheck represents a TCP health check.
type TCPCheck struct {
    Address string
}

// Run executes the TCP health check.
func (tc *TCPCheck) Run() (bool, string) {
    conn, err := net.DialTimeout("tcp", tc.Address, 5*time.Second)
    if err != nil {
        return false, fmt.Sprintf("TCP connection error: %v", err)
    }
    conn.Close()
    return true, "TCP check passed"
}