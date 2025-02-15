package checks

import (
    "fmt"
    "net"
    "time"
)

// TCPCheck tries to open a TCP connection to the given address.
func TCPCheck(address string) (bool, string) {
    conn, err := net.DialTimeout("tcp", address, 5*time.Second)
    if err != nil {
        return false, fmt.Sprintf("TCP connection error: %v", err)
    }
    conn.Close()
    return true, "TCP check passed"
}