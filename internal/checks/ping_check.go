package checks

import (
    "fmt"
    "os/exec"
    "strings"
)

// PingCheck represents a Ping health check.
type PingCheck struct {
    Host string
}

// Run executes the Ping health check.
func (pc *PingCheck) Run() (bool, string) {
    out, err := exec.Command("ping", "-c", "1", "-W", "5", pc.Host).CombinedOutput()
    if err != nil {
        return false, fmt.Sprintf("Ping error: %v", err)
    }
    if strings.Contains(string(out), "1 packets received") {
        return true, "Ping check passed"
    }
    return false, "Ping check failed"
}