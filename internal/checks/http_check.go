package checks

import (
    "fmt"
    "net/http"
    "time"
)

// HTTPCheck performs a basic GET request to verify service availability.
func HTTPCheck(url string, answerPresent bool) (isHealthy bool, message string) {
    client := http.Client{Timeout: 5 * time.Second}
    resp, err := client.Get(url)
    if err != nil {
        return false, fmt.Sprintf("HTTP error: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusOK {
        // Optionally parse body to check certain content
        if answerPresent {
            // If needed, read the body or check a header
        }
        return true, "HTTP check passed"
    }
    return false, fmt.Sprintf("HTTP check failed with status %d", resp.StatusCode)
}