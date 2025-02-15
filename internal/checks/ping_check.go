package checks

import (
    "fmt"
    "os/exec"
    "strings"
)

// PingCheck uses system "ping" command. In production, consider a pure Go approach or library.
func PingCheck(host string) (bool, string) {
    out, err := exec.Command("ping", "-c", "1", "-W", "5", host).CombinedOutput()
    if err != nil {
        return false, fmt.Sprintf("Ping error: %v", err)
    }
    if strings.Contains(string(out), "1 packets received") {
        return true, "Ping check passed"
    }
    return false, "Ping check failed"
}