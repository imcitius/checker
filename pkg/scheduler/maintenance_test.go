package scheduler

import (
	"container/heap"
	"testing"
	"time"

	"checker/pkg/models"
)

// newTestScheduler creates a scheduler with a buffered worker pool for testing.
// Check tracker.jobsReceived() after processNextCheck to see if a job was submitted.
func newTestScheduler() (*Scheduler, *testTracker) {
	tracker := &testTracker{
		jobs: make(chan models.CheckDefinition, 10),
	}
	wp := &WorkerPool{
		workers: 1,
		jobs:    tracker.jobs,
		quit:    make(chan struct{}),
	}
	s := &Scheduler{
		workerPool: wp,
		checkHeap:  &CheckHeap{},
		checkMap:   make(map[string]*CheckItem),
	}
	heap.Init(s.checkHeap)
	return s, tracker
}

type testTracker struct {
	jobs chan models.CheckDefinition
}

// jobsReceived returns the number of jobs in the buffered channel.
func (tt *testTracker) jobsReceived() int {
	return len(tt.jobs)
}

func TestProcessNextCheck_SkipsDuringMaintenanceWindow(t *testing.T) {
	s, tracker := newTestScheduler()

	// Create a check with a maintenance window in the future
	futureTime := time.Now().Add(1 * time.Hour)
	item := &CheckItem{
		CheckDef: models.CheckDefinition{
			UUID:             "maint-check",
			Name:             "Maintained Check",
			Enabled:          true,
			Duration:         "1m",
			MaintenanceUntil: &futureTime,
		},
		NextRun: time.Now().Add(-1 * time.Second), // overdue
	}
	s.checkMap[item.CheckDef.UUID] = item
	heap.Push(s.checkHeap, item)

	// Process the check
	s.processNextCheck()

	// The check should NOT have been submitted to the worker pool
	if tracker.jobsReceived() != 0 {
		t.Error("Expected check in maintenance window to be skipped, but it was submitted")
	}

	// The check should still be in the heap for next scheduling
	if s.checkHeap.Len() != 1 {
		t.Errorf("Expected check to be rescheduled in heap, got heap length %d", s.checkHeap.Len())
	}
}

func TestProcessNextCheck_ExecutesAfterMaintenanceExpires(t *testing.T) {
	s, tracker := newTestScheduler()

	// Create a check with a maintenance window in the past (expired)
	pastTime := time.Now().Add(-1 * time.Hour)
	item := &CheckItem{
		CheckDef: models.CheckDefinition{
			UUID:             "expired-maint-check",
			Name:             "Expired Maintenance Check",
			Enabled:          true,
			Duration:         "1m",
			MaintenanceUntil: &pastTime,
		},
		NextRun: time.Now().Add(-1 * time.Second), // overdue
	}
	s.checkMap[item.CheckDef.UUID] = item
	heap.Push(s.checkHeap, item)

	// Process the check
	s.processNextCheck()

	// The check SHOULD have been submitted
	if tracker.jobsReceived() != 1 {
		t.Error("Expected check with expired maintenance window to be submitted, but it was skipped")
	}
}

func TestProcessNextCheck_ExecutesWithNoMaintenanceWindow(t *testing.T) {
	s, tracker := newTestScheduler()

	// Create a check with no maintenance window
	item := &CheckItem{
		CheckDef: models.CheckDefinition{
			UUID:     "normal-check",
			Name:     "Normal Check",
			Enabled:  true,
			Duration: "1m",
		},
		NextRun: time.Now().Add(-1 * time.Second), // overdue
	}
	s.checkMap[item.CheckDef.UUID] = item
	heap.Push(s.checkHeap, item)

	// Process the check
	s.processNextCheck()

	// The check SHOULD have been submitted
	if tracker.jobsReceived() != 1 {
		t.Error("Expected check without maintenance window to be submitted, but it was skipped")
	}
}
