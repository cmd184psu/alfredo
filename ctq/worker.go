package ctq

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	lockDuration = 5 * time.Minute // Lock expires after 5 minutes
	pollInterval = 5 * time.Second // Check for new tasks every 5 seconds
)

// Worker processes tasks from the queue
type Worker struct {
	db       *DB
	executor *TaskExecutor
	workerID string
	stopChan chan struct{}
}

func NewWorker(db *DB, workerID string) *Worker {
	return &Worker{
		db:       db,
		executor: NewTaskExecutor(db, workerID),
		workerID: workerID,
		stopChan: make(chan struct{}),
	}
}

// Start begins the worker loop
func (w *Worker) Start() error {
	fmt.Printf("[%s] Worker starting...\n", w.workerID)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Cleanup expired locks on startup
	if err := w.db.CleanupExpiredLocks(); err != nil {
		fmt.Printf("[%s] Warning: failed to cleanup expired locks: %v\n", w.workerID, err)
	}

	for {
		select {
		case <-w.stopChan:
			fmt.Printf("[%s] Worker stopping...\n", w.workerID)
			return nil
		case <-sigChan:
			fmt.Printf("[%s] Received shutdown signal\n", w.workerID)
			return nil
		case <-ticker.C:
			if err := w.processNext(); err != nil {
				fmt.Printf("[%s] Error processing task: %v\n", w.workerID, err)
			}
		}
	}
}

// Stop signals the worker to stop
func (w *Worker) Stop() {
	close(w.stopChan)
}

// processNext gets the next task and processes it
func (w *Worker) processNext() error {
	// Check if queue is paused
	if w.db.IsQueuePaused() {
		// Queue is paused, skip processing
		return nil
	}

	// Cleanup expired locks
	if err := w.db.CleanupExpiredLocks(); err != nil {
		return fmt.Errorf("failed to cleanup locks: %w", err)
	}

	// Get next task
	twe, err := w.db.GetNextTask()
	if err != nil {
		panic("failed to get next task: " + err.Error())
//		return fmt.Errorf("failed to get next task: %w", err)
	}

	if twe == nil {
		// No tasks available
		return nil
	}

	task := twe.Task

	// Try to acquire lock
	acquired, err := w.db.AcquireLock(task.ID, w.workerID, lockDuration)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !acquired {
		// Someone else got the lock
		fmt.Printf("[%s] Task %s already locked by another worker\n", w.workerID, task.Name)
		return nil
	}

	// Ensure lock is released
	defer func() {
		if err := w.db.ReleaseLock(task.ID, w.workerID); err != nil {
			fmt.Printf("[%s] Warning: failed to release lock for task %s: %v\n",
				w.workerID, task.Name, err)
		}
	}()

	// Execute the task
	execErr := w.executor.Execute(twe)

	// Task execution complete (error or success)
	// The executor has already recorded the result
	if execErr != nil {
		// Check if we should retry
		retryCount := 0
		if twe.LastExecution != nil {
			retryCount = twe.LastExecution.RetryCount
		}

		if retryCount >= task.MaxRetries {
			fmt.Printf("[%s] Task %s failed after %d retries, giving up\n",
				w.workerID, task.Name, retryCount)

			// If not in requeue mode, disable the task
			if !task.ShouldRequeue() {
				if err := w.db.EnableTask(task.Name, false); err != nil {
					fmt.Printf("[%s] Warning: failed to disable task %s: %v\n",
						w.workerID, task.Name, err)
				}
			}
		}
	}

	return nil
}
