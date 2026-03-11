package scheduler

import (
	"sync"

	"checker/internal/db"
	"checker/internal/models"

	"github.com/sirupsen/logrus"
)

// WorkerPool manages a pool of workers to execute checks
type WorkerPool struct {
	workers      int
	jobs         chan models.CheckDefinition
	wg           sync.WaitGroup
	repo         db.Repository
	slackAlerter *SlackAlerter
	quit         chan struct{}
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int, repo db.Repository, slackAlerter *SlackAlerter) *WorkerPool {
	return &WorkerPool{
		workers:      workers,
		jobs:         make(chan models.CheckDefinition, workers*2), // Buffer slightly
		repo:         repo,
		slackAlerter: slackAlerter,
		quit:         make(chan struct{}),
	}
}

// Start starts the workers
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
	logrus.Infof("Started worker pool with %d workers", wp.workers)
}

// Stop stops the workers and waits for them to finish
func (wp *WorkerPool) Stop() {
	close(wp.quit)
	close(wp.jobs) // Close jobs channel to signal workers to finish
	wp.wg.Wait()
	logrus.Info("Worker pool stopped")
}

// Submit submits a check to be executed
func (wp *WorkerPool) Submit(check models.CheckDefinition) {
	select {
	case wp.jobs <- check:
		// Job submitted
	case <-wp.quit:
		// Pool is stopping, ignore
		logrus.Warn("Worker pool stopping, dropping check submission")
	default:
		// Channel full, this implies the system is overloaded.
		// For a scheduler, we probably shouldn't block indefinitely in the main loop or we delay other checks.
		// But dropping it means missing a scheduled run.
		// For now, let's log a warning and block since reliability is generally preferred over strict timing in this context,
		// or maybe we should increase buffer/workers.
		// A better approach for scalability: if full, maybe spawn a temporary goroutine or just block (backpressure).
		// Let's block for now to apply backpressure to the scheduler loop.
		wp.jobs <- check
	}
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	// logrus.Debugf("Worker %d started", id)

	for {
		select {
		case check, ok := <-wp.jobs:
			if !ok {
				return
			}
			if err := executeCheck(wp.repo, check, wp.slackAlerter); err != nil {
				logrus.Errorf("Worker %d: Error executing check %s: %v", id, check.UUID, err)
			}
		case <-wp.quit:
			return
		}
	}
}
