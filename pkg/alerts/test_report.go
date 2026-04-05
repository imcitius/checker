package alerts

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// testReportConfig holds the configuration for a test_report alert channel.
type testReportConfig struct {
	OutputFile string `json:"output_file"` // required: path to NDJSON output file
}

// testReportEntry is the NDJSON line written for each alert or recovery.
type testReportEntry struct {
	CheckName  string `json:"check_name"`
	CheckType  string `json:"check_type"`
	Project    string `json:"project"`
	Status     string `json:"status"`      // "pass" or "fail"
	Message    string `json:"message"`
	DurationMs int64  `json:"duration_ms"`
	Timestamp  string `json:"timestamp"`   // RFC3339
}

// TestReportAlerter implements the Alerter interface by appending NDJSON lines to a file.
type TestReportAlerter struct {
	config testReportConfig
	mu     sync.Mutex
}

func (a *TestReportAlerter) Type() string { return "test_report" }

func (a *TestReportAlerter) SendAlert(p AlertPayload) error {
	entry := testReportEntry{
		CheckName:  p.CheckName,
		CheckType:  p.CheckType,
		Project:    p.Project,
		Status:     "fail",
		Message:    p.Message,
		DurationMs: 0,
		Timestamp:  p.Timestamp.Format(time.RFC3339),
	}
	return a.appendEntry(entry)
}

func (a *TestReportAlerter) SendRecovery(p RecoveryPayload) error {
	entry := testReportEntry{
		CheckName:  p.CheckName,
		CheckType:  p.CheckType,
		Project:    p.Project,
		Status:     "pass",
		Message:    "",
		DurationMs: 0,
		Timestamp:  p.Timestamp.Format(time.RFC3339),
	}
	return a.appendEntry(entry)
}

// appendEntry marshals the entry to JSON and appends it as a single line to the output file.
func (a *TestReportAlerter) appendEntry(entry testReportEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("test_report: marshal entry: %w", err)
	}
	data = append(data, '\n')

	a.mu.Lock()
	defer a.mu.Unlock()

	f, err := os.OpenFile(a.config.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("test_report: open file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("test_report: write entry: %w", err)
	}
	return nil
}

func newTestReportAlerter(raw json.RawMessage) (Alerter, error) {
	var cfg testReportConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parsing test_report config: %w", err)
	}
	if cfg.OutputFile == "" {
		return nil, fmt.Errorf("test_report requires output_file")
	}
	return &TestReportAlerter{config: cfg}, nil
}

func init() {
	RegisterAlerter("test_report", newTestReportAlerter)
}
