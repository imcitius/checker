package alerts

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestTestReportAlerter_SendAlert(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "results.ndjson")

	alerter := &TestReportAlerter{
		config: testReportConfig{OutputFile: outFile},
	}

	ts := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	err := alerter.SendAlert(AlertPayload{
		CheckName: "api-health",
		CheckType: "http",
		Project:   "my-project",
		Message:   "connection refused",
		Severity:  "critical",
		Timestamp: ts,
	})
	if err != nil {
		t.Fatalf("SendAlert returned error: %v", err)
	}

	entries := readNDJSON(t, outFile)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.CheckName != "api-health" {
		t.Errorf("expected check_name 'api-health', got %q", e.CheckName)
	}
	if e.CheckType != "http" {
		t.Errorf("expected check_type 'http', got %q", e.CheckType)
	}
	if e.Project != "my-project" {
		t.Errorf("expected project 'my-project', got %q", e.Project)
	}
	if e.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", e.Status)
	}
	if e.Message != "connection refused" {
		t.Errorf("expected message 'connection refused', got %q", e.Message)
	}
	if e.Timestamp != "2026-04-05T12:00:00Z" {
		t.Errorf("expected timestamp '2026-04-05T12:00:00Z', got %q", e.Timestamp)
	}
}

func TestTestReportAlerter_SendRecovery(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "results.ndjson")

	alerter := &TestReportAlerter{
		config: testReportConfig{OutputFile: outFile},
	}

	ts := time.Date(2026, 4, 5, 12, 5, 0, 0, time.UTC)
	err := alerter.SendRecovery(RecoveryPayload{
		CheckName: "db-check",
		CheckType: "tcp",
		Project:   "backend",
		Timestamp: ts,
	})
	if err != nil {
		t.Fatalf("SendRecovery returned error: %v", err)
	}

	entries := readNDJSON(t, outFile)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", e.Status)
	}
	if e.Message != "" {
		t.Errorf("expected empty message for recovery, got %q", e.Message)
	}
	if e.CheckName != "db-check" {
		t.Errorf("expected check_name 'db-check', got %q", e.CheckName)
	}
}

func TestTestReportAlerter_AppendsMultipleEntries(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "results.ndjson")

	alerter := &TestReportAlerter{
		config: testReportConfig{OutputFile: outFile},
	}

	ts := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)

	if err := alerter.SendAlert(AlertPayload{
		CheckName: "check-1",
		CheckType: "http",
		Project:   "proj",
		Message:   "err1",
		Timestamp: ts,
	}); err != nil {
		t.Fatalf("SendAlert 1: %v", err)
	}

	if err := alerter.SendAlert(AlertPayload{
		CheckName: "check-2",
		CheckType: "tcp",
		Project:   "proj",
		Message:   "err2",
		Timestamp: ts,
	}); err != nil {
		t.Fatalf("SendAlert 2: %v", err)
	}

	if err := alerter.SendRecovery(RecoveryPayload{
		CheckName: "check-1",
		CheckType: "http",
		Project:   "proj",
		Timestamp: ts,
	}); err != nil {
		t.Fatalf("SendRecovery: %v", err)
	}

	entries := readNDJSON(t, outFile)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].CheckName != "check-1" || entries[0].Status != "fail" {
		t.Errorf("entry 0: expected check-1/fail, got %s/%s", entries[0].CheckName, entries[0].Status)
	}
	if entries[1].CheckName != "check-2" || entries[1].Status != "fail" {
		t.Errorf("entry 1: expected check-2/fail, got %s/%s", entries[1].CheckName, entries[1].Status)
	}
	if entries[2].CheckName != "check-1" || entries[2].Status != "pass" {
		t.Errorf("entry 2: expected check-1/pass, got %s/%s", entries[2].CheckName, entries[2].Status)
	}
}

func TestTestReportAlerter_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "results.ndjson")

	alerter := &TestReportAlerter{
		config: testReportConfig{OutputFile: outFile},
	}

	ts := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_ = alerter.SendAlert(AlertPayload{
				CheckName: "concurrent-check",
				CheckType: "http",
				Project:   "proj",
				Message:   "error",
				Timestamp: ts,
			})
		}()
	}
	wg.Wait()

	entries := readNDJSON(t, outFile)
	if len(entries) != n {
		t.Errorf("expected %d entries from concurrent writes, got %d", n, len(entries))
	}
}

func TestTestReportAlerter_InvalidPath(t *testing.T) {
	alerter := &TestReportAlerter{
		config: testReportConfig{OutputFile: "/nonexistent/dir/results.ndjson"},
	}

	err := alerter.SendAlert(AlertPayload{
		CheckName: "test",
		CheckType: "http",
		Project:   "proj",
		Message:   "error",
		Timestamp: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}

func TestNewTestReportAlerter_Valid(t *testing.T) {
	cfg := json.RawMessage(`{"output_file":"/tmp/test-results.ndjson"}`)
	a, err := NewAlerter("test_report", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tra, ok := a.(*TestReportAlerter)
	if !ok {
		t.Fatalf("expected *TestReportAlerter, got %T", a)
	}
	if tra.config.OutputFile != "/tmp/test-results.ndjson" {
		t.Errorf("unexpected output_file: %q", tra.config.OutputFile)
	}
	if tra.Type() != "test_report" {
		t.Errorf("expected Type() 'test_report', got %q", tra.Type())
	}
}

func TestNewTestReportAlerter_MissingOutputFile(t *testing.T) {
	_, err := NewAlerter("test_report", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for missing output_file, got nil")
	}
}

func TestNewTestReportAlerter_InvalidJSON(t *testing.T) {
	_, err := NewAlerter("test_report", json.RawMessage(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// readNDJSON reads all NDJSON entries from the given file.
func readNDJSON(t *testing.T, path string) []testReportEntry {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open %s: %v", path, err)
	}
	defer f.Close()

	var entries []testReportEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e testReportEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			t.Fatalf("failed to unmarshal NDJSON line: %v", err)
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}
	return entries
}
